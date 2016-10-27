package main

import (
	"fmt"
	"strconv"
	"math/rand"
	"strings"
	"net/http"
	"io/ioutil"
	"os/exec"
	"github.com/bwmarrin/discordgo"
	"time"
	"os"
	"encoding/json"
)

var token string
var botID string
var key string
var showOnce map[string]bool = make(map[string]bool)
var playing []string =[]string {"WITH THE FATE OF THE UNIVERSE", "jesus", "with his feet", "in the woods", "with other bears"}
var responses []string = []string {":anger:`Rawr?`", "How did i end up in this thing?", "I will eat you when i get my powers back"}

//add commands to add responses
//add commands to add new playing options

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
	//
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

func ready(s *discordgo.Session, event *discordgo.Ready) {
	if botID == ""{
		botID = s.State.User.ID
	}
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

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == botID{
		return	
	}
	//permissions
	var has bool = false
	cha, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fmt.Println(err)
		s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, how dare you crash`")
		return
	}else if !showOnce[cha.GuildID]{
		role := serverHasRole(s, cha.GuildID)
		if role != ""{
			mem, err := s.State.Member(cha.GuildID, m.Author.ID)
			if err!= nil {
				fmt.Println(err)
				s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, how dare you crash`")
				return
			}else{
				for i:= 0; i < len (mem.Roles); i++{
					if mem.Roles[i] == role{
						has = true
					}
				}
			}
		} else{
			fmt.Println("server id: ",cha.GuildID," Does not have permission, all requests run")
			showOnce[cha.GuildID] = true
			has = true
		}
	}
	mentions := m.Mentions
	if len(mentions) > 0 {
		if mentions[0].ID == botID {
			if len(m.Content) < 22{
					if !has {
						s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
						return
					}
				rand := rand.Intn(2100)
				s.ChannelMessageSend(m.ChannelID, responses[rand%len(responses)] )
			}else{
				term := strings.ToLower(strings.TrimSpace(strings.SplitAfter(m.Content,">")[1]))
				if strings.Contains(term, "bear"){
					if !has {
						s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
						return
					}
					term = strings.Replace(term, " ", "%20",-1)
					s.ChannelMessageSend(m.ChannelID, getImage(term))
				}else if  strings.Contains(term, "cpu"){
					if !has {
						s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
						return
					}
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("CPU usage is %f%%", getCPUUsage(1000)))
				}else{
					if !has {
						s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
						return
					}
					s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, how dare you not search for bears`")
				}
			}
		}
	}else if m.Content == "🐻"{
		if !has {
			s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
			return
		}
		s.ChannelMessageSend(m.ChannelID, getImage("bear"))
	}else if strings.ToLower(m.Content) == "bear"{
		if !has {
			s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Rawr")
	} else if strings.HasPrefix(strings.ToLower(m.Content), "set playing"){
		if !has {
			s.ChannelMessageSend(m.ChannelID, ":anger: `Rawr, you dont have permission to do that`")
			return
		}
		setBotStatus(s, strings.SplitAfter(m.Content,"set playing")[1])
	}else{
		rand := rand.Intn(21)
		if rand > 19  {
			s.ChannelMessageSend(m.ChannelID, ":bear:")
		}
	}
}

func getCPUUsage(sleepTime int) (total float64) {
    contents, err := ioutil.ReadFile("/proc/stat")
	if err == nil {
		var idle, total [2]uint64
		lines := strings.Split(string(contents), "\n")
		for j:= 0; j < 2; j++ {
			for _, line := range(lines) {
				fields := strings.Fields(line)
				if fields[0] == "cpu" {
					for i := 1; i < len(fields); i++ {
						val, err := strconv.ParseUint(fields[i], 10, 64)
						if err != nil {
							fmt.Println("Error: ", i, fields[i], err)
						}
						total[j] += val
						if i == 4 { 
							idle[j] = val
						}
					}
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
	}
	return ":anger: `NO GOOGLE API KEY`"
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