package main

import (
	"fmt"
	"strconv"
	"math/rand"
	"strings"
	"os/exec"
	"net/http"
	"io/ioutil"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/dgvoice"
	"time"
	"os"
	"encoding/json"
	"regexp"
	"errors"
	"github.com/rylio/ytdl"
)

var token string
var botID string
var key string
var serverRole map[string]ServerRole = make(map[string]ServerRole)
var playing []string =[]string {"WITH THE FATE OF THE UNIVERSE", "jesus", "with his feet", "in the woods", "with other bears"}
var responses []string = []string {":anger:`Rawr?`", "How did i end up in this thing?", "I will eat you when i get my powers back"}
var dgv *discordgo.VoiceConnection = nil
//add a queue for the bot to play
//add a command to clear the queue
//add a command to skip current song

func init() {
	readConfig()
}

func readConfig(){
	if _, err := os.Stat("bear.config"); os.IsNotExist(err) {
		creds := &Credentials {Playing : playing, Responses : responses, Token : "", GoogleAPIKey : "" }
		json, err := json.Marshal(creds)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = ioutil.WriteFile("bear.config", []byte(json), os.ModeDevice)
		if err != nil {
			fmt.Println(err)
			return
		}
	}else {
		contents, err := ioutil.ReadFile("bear.config")
		if err != nil {
			fmt.Println(err)
			return
		}
		var config Credentials
		err = json.Unmarshal(contents, &config)
		if err != nil{
			fmt.Println("Problem parsing config file",err)
			return
		}
		playing = config.Playing
		responses = config.Responses
		token = config.Token
		key = config.GoogleAPIKey
		if len(token) > 0 {
			if !strings.HasPrefix(strings.ToLower(token), "bot" ) {
				token = fmt.Sprintf("Bot %s", token)
			}
		}
	}
}

func setBotStatus(session *discordgo.Session, status string){
	err := session.UpdateStatus(0, status)
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	if token != ""{
		dg, err := discordgo.New(token)
		if err != nil {
			fmt.Println("Error creating Discord session: ", err)
			return
		}
		dg.AddHandler(ready)
		dg.AddHandler(messageCreate)
		err = dg.Open()
		if err != nil {
			fmt.Println("Error opening Discord session: ", err)
			return
		}
		fmt.Println("Bot is now running.  Press CTRL-C to exit.")
		<-make(chan struct{})
	}
	fmt.Println("Add Bot ID to config file.")
	return
}

func getBotId(s *discordgo.Session){
if botID == ""{
		if s.State != nil {
			if s.State.User != nil {
				if s.State.User.ID != "" {
					botID = s.State.User.ID
				}
			}
		}
	}
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	getBotId(s)
	go changeStatus(s)
}

func changeStatus(s *discordgo.Session){
	rand := rand.Intn(2100)
	setBotStatus(s, playing[rand%len(playing)])
	sleeptime := (1000 * rand) + 600
	time.Sleep( time.Duration(sleeptime) * time.Millisecond)
	changeStatus(s)
}

func serverHasRole(s *discordgo.Session, guildID string) (role string){
	for i:=0; i < len(s.State.Guilds); i++ {
		if s.State.Guilds[i].ID == guildID{
			for j:=0; j < len (s.State.Guilds[i].Roles); j++{
				if s.State.Guilds[i].Roles[j].Name == "Bots"{
					return s.State.Guilds[i].Roles[j].ID
				}
			}
		}
	}
	return ""
}

func userHasRole (guild, user, role string, s *discordgo.Session ) (has bool, err error){
	mem, err := s.State.Member(guild, user)
	if err!= nil {
		return false, err
	}else{
		for i:= 0; i < len (mem.Roles); i++{
			if mem.Roles[i] == role{
				return true, nil
			}
		}
	}
	return false, nil
}

func runCmd(s *discordgo.Session, channel, user string)(run bool, err error){
	has := false
	cha, err := s.State.Channel(channel)
	if err != nil {
		return false, err
	}
	role := serverRole[cha.GuildID]
	if len(role.ID) == 0 && !role.shownError {
		roleID := serverHasRole(s, cha.GuildID)
		if roleID != ""{
			serverRole[cha.GuildID] = ServerRole {ID : roleID, shownError : false }
			has, err = userHasRole(cha.GuildID, user, roleID, s)
			if err != nil {
				return false, err
			}
		} else{
			fmt.Println("server id: ",cha.GuildID," Does not have permission, all requests run")
			serverRole[cha.GuildID] = ServerRole {ID : "", shownError : true }
			has = true
		}
	}else{
		has, err = userHasRole(cha.GuildID, user, role.ID, s)
		if err != nil {
			return false, err
		}
	}
	return has, nil
}

func checkForUser(s *discordgo.Session, channelID, authorID string) (string, string){
	c, err := s.State.Channel(channelID)
	if err != nil {
		return "", ""
	}
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		return "", ""
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == authorID {
			return vs.ChannelID, g.ID
		}
	}
	return "", ""
}

func isAYoutubeLink(url string) (bool, error){
	matched, err := regexp.MatchString("^https?://.*(?:youtu(.be)?(be.com)?/|v/|u/\\w/|embed/|watch?v=)([^#&?]*).*$", url)
	return matched , err
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if botID == "" {
		getBotId(s)
	}
	if botID == "" || m.Author.ID == botID{
		return	
	}
	//is a command
	if strings.HasPrefix(strings.ToLower(m.Content),"!bb") || strings.HasPrefix(strings.ToLower(m.Content),"!bearbot") {
		//permissions
		has, err := runCmd(s, m.ChannelID, m.Author.ID)
		if err != nil {
			fmt.Println(err)
			s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr`")
			return
		}
		if !has {
			s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
			return
		}
		var cmd string
		if strings.HasPrefix(strings.ToLower(m.Content),"!bb"){
			if len(m.Content) > 4 {
				cmd = m.Content[4:]
			} else {
				cmd = ""
			}
		}else{
			if len(m.Content) > 9 {
				cmd = m.Content[9:]
			} else {
				cmd = ""
			}
		}
		if len(cmd) > 0 {
			switch {
			case cmd == "bear":
				s.ChannelMessageSend(m.ChannelID, "Rawr")
			case cmd =="cpu":
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("CPU usage is %f%%", getCPUUsage(1000)))
			case cmd == "🐻":
				s.ChannelMessageSend(m.ChannelID, getImage("bear"))
			case strings.HasPrefix(strings.ToLower(cmd), "set playing"):
				setBotStatus(s, strings.SplitAfter(m.Content,"set playing")[1])
			case strings.HasPrefix(strings.ToLower(cmd), "img"):
				term := strings.ToLower(strings.TrimSpace(cmd[4:]))
				term = strings.Replace(term, " ", "+",-1)
				s.ChannelMessageSend(m.ChannelID, getImage(term))
			case strings.HasPrefix(strings.ToLower(cmd), "image"):
				term := strings.ToLower(strings.TrimSpace(cmd[6:]))
				term = strings.Replace(term, " ", "+",-1)
				s.ChannelMessageSend(m.ChannelID, getImage(term))
			case strings.HasPrefix(strings.ToLower(cmd), "youtube"):
				term := strings.ToLower(strings.TrimSpace(cmd[8:]))
				term = strings.Replace(term, " ", "+",-1)
				s.ChannelMessageSend(m.ChannelID, getYTVid(term, false))
			case strings.HasPrefix(strings.ToLower(cmd),"ytp"):
				channelId, guildID := checkForUser(s, m.ChannelID, m.Author.ID)
				if channelId != "" {
					matched, err := isAYoutubeLink(cmd[4:])
					if err == nil {
						if matched{
							go playVideoSound(s,guildID,channelId,cmd[4:])
							s.ChannelMessageDelete(channelId, m.ID)
						} else{
							term := strings.ToLower(strings.TrimSpace(cmd[4:]))
							term = strings.Replace(term, " ", "+",-1)
							url := getYTVid(term, false)
							go playVideoSound(s,guildID,channelId,url)
						}
					}
				}else{
					s.ChannelMessageSend(m.ChannelID, ":anger:`Not in voice channel`")
				}
			case strings.ToLower(cmd) == "stop":
				stopPlaying()
			case strings.HasPrefix(strings.ToLower(cmd), "ytr"):
				term := strings.ToLower(strings.TrimSpace(cmd[4:]))
				term = strings.Replace(term, " ", "+",-1)
				s.ChannelMessageSend(m.ChannelID, getYTVid(term, true))			
			case strings.HasPrefix(strings.ToLower(cmd), "yt"):
				term := strings.ToLower(strings.TrimSpace(cmd[3:]))
				term = strings.Replace(term, " ", "+",-1)
				s.ChannelMessageSend(m.ChannelID, getYTVid(term, false))
			case strings.HasPrefix(strings.ToLower(cmd),"allow"):
				mentions := m.Mentions
				if len(mentions) > 0 {
					for i:=0; i < len(mentions); i++{
						err := modifyUser(s, true, mentions[i], m.ChannelID)
						if err != nil {
							fmt.Println(err)
							continue
						}
					}
				} else {
					s.ChannelMessageSend(m.ChannelID, ":anger:`No user to allow`")
				}
			case strings.HasPrefix(strings.ToLower(cmd),"remove"):
				mentions := m.Mentions
				if len(mentions) > 0 {
					for i:=0; i < len(mentions); i++{
						err := modifyUser(s, false, mentions[i], m.ChannelID)
						if err != nil {
							fmt.Println(err)
							continue
						}
					}
				} else {
					s.ChannelMessageSend(m.ChannelID, ":anger:`No user to remove`")
				}
			//add more commands to accept new responses and accept new statuses
			default:
				s.ChannelMessageSend(m.ChannelID, ":anger:`Not a command`")
			}
		} else{
			s.ChannelMessageSend(m.ChannelID, ":anger:`Not a command`")
			//blank command
			//list commands
		}
	}else{
		rand := rand.Intn(21)
		if rand > 19  {
			s.ChannelMessageSend(m.ChannelID, ":bear:")
		}
	}
}

func getStream(url string) (out string){
	info, err := ytdl.GetVideoInfo(url)
	if err!= nil {
		return ""
	}
	formats := info.Formats
	var vid, noVid ytdl.Format
	for i:= 0; i< len(formats); i++{
		if formats[i].VideoEncoding== ""{
			if noVid.AudioBitrate < formats[i].AudioBitrate{
				noVid = formats[i]
			}
		}else{
			if vid.AudioBitrate < formats[i].AudioBitrate{
				vid = formats[i]
			}
		}
	}
	if noVid.AudioBitrate > 0{
		downloadURL, err := info.GetDownloadURL(noVid)
		if err != nil {
			return ""
		}
		return downloadURL.String()
	} else {
		downloadURL, err := info.GetDownloadURL(vid)
		if err != nil {
			return ""
		}
		return downloadURL.String()
	}
}

func stopPlaying(){
	if dgv != nil {
		dgvoice.KillPlayer()
		dgv = nil
	}
}

func playVideoSound(s *discordgo.Session, guildID, channelID, url string) error {
	vid := getStream(url)
	if vid != "" {
		var err error
		dgv, err = s.ChannelVoiceJoin(guildID, channelID, false, true)
		if err != nil {
			return err
		}
		dgvoice.PlayAudioFile(dgv, vid)
		if dgv != nil {
			dgv.Close()
			dgv = nil
		}
	}
	return nil
}

func modifyUser(s *discordgo.Session, add bool, mention *discordgo.User, ChannelID string) (err error) {
	cha, err := s.State.Channel(ChannelID)
	if err != nil {
		return err
	}
	member, err := s.State.Member(cha.GuildID,mention.ID)
	if err != nil{
		return err
	}
	role := serverHasRole(s, cha.GuildID)
	if add {
		for i:=0; i < len (member.Roles); i++{
			if member.Roles[i] == role {
				return errors.New("already has role")
			}
		}
		member.Roles = append(member.Roles, role)
		err := s.GuildMemberEdit(cha.GuildID, mention.ID, member.Roles)
		return err
	}else{
		hasRole := false
		var roles []string
		for i:=0; i < len (member.Roles); i++{
			if member.Roles[i] == role {
				hasRole = true
				continue
			}
			roles = append(roles, member.Roles[i])
		}
		if !hasRole {
			return errors.New("doesnt have role")
		}
		member.Roles = roles
		err := s.GuildMemberEdit(cha.GuildID, mention.ID, member.Roles)
		return err
	}
	return err
}

func getYTVid(content string, isRand bool) string{
	if key != "" {
		//if strings.Contains(content, "bear") {
			maxRes := 1
			if isRand {
				maxRes = 50
			}
			url :=fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=%d&q=%s&key=%s", maxRes,content, key)
			resp, err := http.Get(url)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			var res video
			err = json.Unmarshal(body, &res)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			if len(res.Items) > 0 {
				item := 0
				if isRand {
					item = rand.Intn(49)+1
				}
				return fmt.Sprintf("https://youtube.com/watch?v=%s", res.Items[item].ID.VideoID) 
			}
			return ":anger: `RAWR NO VIDEOS EXIST`"
		//} else{
		//	return ":anger: `Rawr, how dare you not search for bears`"
		//}
	}
	return ":anger: `NO GOOGLE API KEY`"
}

func getCPUUsage(sleepTime int) (total float64) {
    _, err := ioutil.ReadFile("/proc/stat")
	if err == nil {
		var idle, total [2]uint64
		for j:= 0; j < 2; j++ {
			contents, _ := ioutil.ReadFile("/proc/stat")
			lines := strings.Split(string(contents), "\n")
			for _, line := range(lines) {
				fields := strings.Fields(line)
				if fields[0] == "cpu" {
					for i := 1; i < len(fields); i++ {
						val, err := strconv.ParseUint(fields[i], 10, 64)
						if err != nil {
							fmt.Println("Error: ", i, fields[i], err)
						}
						total[j] += val
						if i == 4 || i == 5  { 
							idle[j] += val
						}
					}
					break
				}
			}
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
		idleTicks := float64(idle[1] - idle[0])
		totalTicks := float64(total[1] - total[0])
		return 100 * (totalTicks - idleTicks) / totalTicks
	} else {
		out, err := exec.Command("wmic", "cpu", "get", "loadpercentage").Output()
		if err != nil {
			return
		}
		output, err := strconv.ParseInt(strings.TrimSpace(strings.Replace(string(out), "LoadPercentage", "", -1)),10,64)
		if err != nil {
			return
		}
		return float64(output)
	}
}

func getImage(content string) string{
	if key != "" {
		//if strings.Contains(content, "bear") {
			rand := rand.Intn(100)
			url :=fmt.Sprintf("https://www.googleapis.com/customsearch/v1?q=%s&filter=1&num=1&start=%d&safe=high&cx=003565045981236897371:jydtkxy12lw&searchType=image&key=%s", content, rand, key)
			resp, err := http.Get(url)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			var res response
			err = json.Unmarshal(body, &res)
			if err != nil {
				fmt.Println(err)
				return ":anger: `RAWR`"
			}
			if len(res.Items) > 0 {
				return res.Items[0].Link
			}
			return ":anger: `RAWR NO IMAGES EXIST`"
		//} else{
		//	return ":anger: `Rawr, how dare you not search for bears`"
		//}
	}
	return ":anger: `NO GOOGLE API KEY`"
}

type ServerRole struct {
	ID string
	shownError bool
}

type response struct {
	Items []struct {
		Link string `json:"link"`
	} `json:"items"`
}

type Credentials struct {
	GoogleAPIKey string   `json:"GoogleApiKey"`
	Playing      []string `json:"Playing"`
	Responses    []string `json:"Responses"`
	Token        string   `json:"Token"`
}

type video struct {
	Items []struct {
		ID   struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}
