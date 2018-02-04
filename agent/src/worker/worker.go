package worker

import (
	"time"
	"io/ioutil"
	"strconv"
	"os"
	"os/exec"
	"syscall"
	"bufio"
	"strings"
	linuxproc "github.com/c9s/goprocinfo/linux"
	"github.com/drael/GOnetstat"
        "github.com/beevik/ntp"

)

type Check struct {
        ConfigLabel string
	Host string
        TimeStamp string
        EpochTime int64
        Command string
        Output string
        Retval int
}

type Shadow struct {
	Username string
	Encpass string
	Lastchg int
	Mindays int 
	Maxdays int 
	Warndays int 
	Inactdays int 
	Expiredays int 
	Flag string
}


func LoadAverage(Label string) (Check, error) {
	loadavg := Check{}

	now := time.Now()
	current_time := time.Now().Local()

	epoch := now.Unix()
	t := current_time.Format("Jan 02 2006 03:04:05")

	data, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		loadavg.ConfigLabel = Label
		loadavg.TimeStamp = t
		loadavg.EpochTime = epoch
		loadavg.Command = "LoadAverage"
		loadavg.Output = "error reading /proc/loadavg: " + err.Error() 
		loadavg.Retval = 1

		return loadavg, err
	}

	loadavg.ConfigLabel = Label
	loadavg.TimeStamp = t
	loadavg.EpochTime = epoch
	loadavg.Command = "LoadAverage"
	loadavg.Output = string(data)
	loadavg.Retval = 0

	return loadavg, nil	
}

func MemUsage(Label string) (Check, error) {
	memusage := Check{}

	now := time.Now()
        current_time := time.Now().Local()

        epoch := now.Unix()
        t := current_time.Format("Jan 02 2006 03:04:05")


	data, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		memusage.ConfigLabel = Label
		memusage.TimeStamp = t
		memusage.EpochTime = epoch
		memusage.Command = "MemUsage"
		memusage.Output = "error reading /proc/meminfo: " + err.Error() 
		memusage.Retval = 1

		return memusage, err
	}

	memused := data.MemTotal - data.MemAvailable
	musedperc := float64((float64(memused) / float64(data.MemTotal)) * 100)
	memusedperc := strconv.FormatFloat(musedperc, 'f', 0, 64) 

        memusage.ConfigLabel = Label
        memusage.TimeStamp = t
        memusage.EpochTime = epoch
	memusage.Command = "MemUsage"
	memusage.Output = strconv.FormatUint(memused, 10) + "/" + strconv.FormatUint(data.MemTotal, 10) + "/" + memusedperc + "%%"
	memusage.Retval = 0

	return memusage, nil
}

func CheckDiskUsage(Label string, Path string) (Check, error) {
	disk := Check{}

	var stat syscall.Statfs_t
	syscall.Statfs(Path, &stat)	

	disktotal := stat.Blocks * uint64(stat.Bsize)
	diskfree := stat.Bavail * uint64(stat.Bsize)
	diskused := disktotal - diskfree
	dskusedperc := float64((float64(diskused) / float64(disktotal)) * 100)
	diskusedperc := strconv.FormatFloat(dskusedperc, 'f', 0, 64)

	inodetotal := stat.Files
	inodefree := stat.Ffree
	inodeused := inodetotal - inodefree
	indeusedperc := float64((float64(inodeused) / float64(inodetotal)) * 100)
	inodeusedperc := strconv.FormatFloat(indeusedperc, 'f', 0, 64)

	strdisktotal := strconv.FormatUint(disktotal, 10)
	strdiskfree := strconv.FormatUint(diskfree, 10)
	strdiskused := strconv.FormatUint(diskused, 10)
	
	strinodetotal := strconv.FormatUint(inodetotal, 10)
	strinodefree := strconv.FormatUint(inodefree, 10)
	strinodeused := strconv.FormatUint(inodeused, 10)

	now := time.Now()
	current_time := time.Now().Local()

	epoch := now.Unix()
	t := current_time.Format("Jan 02 2006 03:04:05")

	disk.ConfigLabel = Path + " " + Label
	disk.TimeStamp = t
	disk.EpochTime = epoch
	disk.Command = "CheckDiskUsage"
	disk.Output = strdisktotal + "|" + strdiskused + "|" + strdiskfree + "|" + diskusedperc + "%%" + "|" + strinodetotal + "|" + strinodeused + "|" + strinodefree + "|" + inodeusedperc + "%%"  
	disk.Retval = 0

	return disk, nil	

}

func CheckPassword(Label string, User string) (Check, error) {
	user := Check{}
	shadow := Shadow{}

	now := time.Now()
	current_time := time.Now().Local()

	epoch := now.Unix()
	t := current_time.Format("Jan 02 2006 03:04:05")


	file, err := os.Open("/etc/shadow")
	if err != nil {
		user.ConfigLabel = Label
		user.TimeStamp = t
		user.EpochTime = epoch
		user.Command = "CheckPassword: [" + User + "]"
		user.Output = "error reading /etc/shadow: " + err.Error() 
		user.Retval = 1

		return user, err
	}
	defer file.Close()


	flag := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if parts[0] == User {
			flag = true
			shadow.Username = parts[0]

			lchg, _ := strconv.Atoi(parts[2])
			mndays, _ := strconv.Atoi(parts[3])
			mxdays, _ := strconv.Atoi(parts[4])
			wndays, _ := strconv.Atoi(parts[5])
			indays, _ := strconv.Atoi(parts[6])
			exdays, _ := strconv.Atoi(parts[7])

			shadow.Lastchg = lchg 
			shadow.Mindays = mndays 
			shadow.Maxdays = mxdays 
			shadow.Warndays = wndays 
			shadow.Inactdays = indays 
			shadow.Expiredays = exdays 
			shadow.Flag = parts[8]
		}
	}


	day := 24 * 3600
	scale := day
	changed := shadow.Lastchg * scale

	expires := ""
	if shadow.Lastchg <= 0 || shadow.Maxdays >= 10000 * (day / scale) || shadow.Maxdays < 0 {
		expires = "never"
	} else {
		iexpires := changed + shadow.Maxdays * scale
		expires = strconv.Itoa(iexpires)
	}


	user.ConfigLabel = Label
	user.TimeStamp = t
	user.EpochTime = epoch
	user.Command = "CheckPassword: [" + User + "]"

	if flag {
		user.Output = expires 
		user.Retval = 0
	} else {
		user.Output = "user doesn't exist"
		user.Retval = 1
	}

	return user, nil	
}

func CheckSSH(Label string) (Check, error) {
	ssh := Check{}
	
	d := GOnetstat.Tcp()

	flag := false
	for _, p := range d {
		if p.State == "LISTEN" {
			if p.Exe == "sshd" {
				flag = true
			}
		}
	}

	now := time.Now()
	current_time := time.Now().Local()

	epoch := now.Unix()
	t := current_time.Format("Jan 02 2006 03:04:05")

	ssh.ConfigLabel =  Label
	ssh.TimeStamp = t
	ssh.EpochTime = epoch
	ssh.Command = "CheckSSH"

	if flag {
		ssh.Output = "SSH is up"
		ssh.Retval = 0
	} else {
		ssh.Output = "SSH is DOWN"
		ssh.Retval = 1
	}

	return ssh, nil	
}

func CheckSwap(Label string) (Check, error) {
	swapusage := Check{}

	now := time.Now()
        current_time := time.Now().Local()

        epoch := now.Unix()
        t := current_time.Format("Jan 02 2006 03:04:05")

	data, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		swapusage.ConfigLabel = Label
		swapusage.TimeStamp = t
		swapusage.EpochTime = epoch
		swapusage.Command = "CheckSwap"
		swapusage.Output = "error reading /proc/meminfo: " + err.Error()
		swapusage.Retval = 1
		return swapusage, err
	}

	swapused := data.SwapTotal - data.SwapFree
	swpusedperc := float64((float64(swapused) / float64(data.SwapTotal)) * 100)
	swapusedperc := strconv.FormatFloat(swpusedperc, 'f', 0, 64) 

        swapusage.ConfigLabel = Label
        swapusage.TimeStamp = t
        swapusage.EpochTime = epoch
	swapusage.Output = strconv.FormatUint(swapused, 10) + "/" + strconv.FormatUint(data.SwapTotal, 10) + "/" + swapusedperc + "%%"
	swapusage.Retval = 0

	return swapusage, nil
}

func CheckNTPSkew(Label string, Server string) (Check, error) {
	ntpskew := Check{}

        now := time.Now()
        current_time := time.Now().Local()

        epoch := now.Unix()
        t := current_time.Format("Jan 02 2006 03:04:05")

        response, err := ntp.Query(Server)
        if err != nil {
		ntpskew.ConfigLabel = Label
		ntpskew.TimeStamp = t
		ntpskew.EpochTime = epoch
		ntpskew.Command = "CheckNTPSkew"
		ntpskew.Output = "error querying ntp server: " + err.Error() 
		ntpskew.Retval = 1
	
		return ntpskew, err
        }

        offset := response.ClockOffset.String() 


        ntpskew.ConfigLabel = Label
        ntpskew.TimeStamp = t
        ntpskew.EpochTime = epoch
	ntpskew.Command = "CheckNTPSkew"
	ntpskew.Output = offset 
	ntpskew.Retval = 0

	return ntpskew, nil
}

func CheckMailQ(Label string) (Check, error) {
	check := Check{}

        now := time.Now()
        current_time := time.Now().Local()

        epoch := now.Unix()
        t := current_time.Format("Jan 02 2006 03:04:05")

        _, err := os.Stat("/var/spool/clientmqueue/")
        if os.IsNotExist(err) {
		check.ConfigLabel = Label
		check.TimeStamp = t
		check.EpochTime = epoch
		check.Command = "CheckMailQ"
		check.Output = "/var/spool/clientmqueue doesn't exist" 
		check.Retval = 1
		return check, err
        }

        files, err := ioutil.ReadDir("/var/spool/clientmqueue/")
        if err != nil {
		check.ConfigLabel = Label
		check.TimeStamp = t
		check.EpochTime = epoch
		check.Command = "CheckMailQ"
		check.Output = "Can't read from /var/spool/clientmqueue/" + err.Error()
		check.Retval = 1
		return check, err
        }

        check.ConfigLabel = Label
        check.TimeStamp = t
        check.EpochTime = epoch
	check.Command = "CheckMailQ"
	check.Output = strconv.Itoa(len(files)) 
	check.Retval = 0

	return check, nil
}

func RunExternal(Label string, pth string) (Check, error) {
	check := Check{}

        now := time.Now()
        current_time := time.Now().Local()

        epoch := now.Unix()
        t := current_time.Format("Jan 02 2006 03:04:05")

	_, err := os.Stat(pth)
	if os.IsNotExist(err) {
		check.ConfigLabel = Label
		check.TimeStamp = t
		check.EpochTime = epoch
		check.Command = pth
		check.Output = "can't find external utility: " + pth 
		check.Retval = 1
		return check, err
	}

	out, err := exec.Command(pth).Output()
	if err != nil {
		check.ConfigLabel = Label
		check.TimeStamp = t
		check.EpochTime = epoch
		check.Command = pth
		check.Output = "failed to run (" + pth + "):" + err.Error()
		check.Retval = 1
		return check, err
	}

        check.ConfigLabel = Label
        check.TimeStamp = t
        check.EpochTime = epoch
	check.Command = pth
	check.Output = "Success: " + string(out) 
	check.Retval = 0

	return check, nil
}
