package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"moarchan/wsframe"
	"github.com/joncody/wsrooms"
	"github.com/vincent-petithory/dataurl"
)

var app *wsframe.App

type Thread struct {
	Type     string           `json:"type"`
	Topic    string           `json:"topic"`
	Name     string           `json:"name"`
	Subject  string           `json:"subject"`
	Options  string           `json:"options"`
	Comment  string           `json:"comment"`
	Replies  map[string]Reply `json:"replies"`
	TaggedBy []interface{}    `json:"taggedBy"`
	Tagging  []interface{}    `json:"tagging"`
	FileInfo
	Unique
}

type Reply struct {
	Type     string   `json:"type"`
	Thread   string   `json:"thread"`
	Topic    string   `json:"topic"`
	Name     string   `json:"name"`
	Options  string   `json:"options"`
	Comment  string   `json:"comment"`
	TaggedBy []string `json:"taggedBy"`
	Tagging  []string `json:"tagging"`
	FileInfo
	Unique
}

type Unique struct {
	Timestamp string `json:"timestamp"`
	Uuid      string `json:"uuid"`
	Hash      string `json:"hash"`
}

type FileInfo struct {
	File       string `json:"file"`
	Name       string `json:"file_name"`
	Path       string `json:"file_path"`
	Mime       string `json:"file_mime"`
	Size       string `json:"file_size"`
	Dimensions string `json:"file_dimensions"`
}

func (f *FileInfo) Process() {
	var config image.Config
	if f.File == "" {
		return
	}
	f.Path = fmt.Sprintf("./static/images/uploads/%s", f.Name)
	fdata, err := dataurl.DecodeString(f.File)
	if err != nil {
		log.Println(err)
		return
	}
	ioutil.WriteFile(f.Path, fdata.Data, 0775)
	saved, err := os.Open(f.Path)
	if err != nil {
		log.Println(err)
		return
	}
	if f.Mime == "image/jpeg" {
		config, err = jpeg.DecodeConfig(saved)
	} else if f.Mime == "image/png" {
		config, err = png.DecodeConfig(saved)
	} else if f.Mime == "image/gif" {
		config, err = gif.DecodeConfig(saved)
	}
	f.Dimensions = fmt.Sprintf("%dx%d", config.Width, config.Height)
	f.File = ""
}

func (u *Unique) Generate() {
	now := time.Now()
	u.Timestamp = fmt.Sprintf("%d/%d/%d(%s)%d:%d:%d", now.Month(), now.Day(), now.Year(), now.Weekday().String()[:3], now.Hour(), now.Minute(), now.Second())
	id, err := uuid.NewRandom()
	if err != nil {
		log.Println(err)
		return
	}
	u.Uuid = id.String()
	u.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s%s", u.Timestamp, u.Uuid))))[:9]
}

func SendMessage(conn *wsrooms.Conn, msg *wsrooms.Message) {
	bmsg := msg.Bytes()
	conn.Emit(msg)
	conn.Send <- bmsg
}

func threadHandler(conn *wsrooms.Conn, msg *wsrooms.Message) {
	var thread Thread
	err := json.Unmarshal(msg.Payload, &thread)
	if err != nil {
		log.Println(err)
		return
	}
	thread.Unique.Generate()
	thread.FileInfo.Process()
	payload, err := json.Marshal(&thread)
	if err != nil {
		log.Println(err)
		return
	}
	if err := app.InsertRow(thread.Topic, thread.Unique.Hash, string(payload)); err != nil {
		return
	}
	response := wsrooms.ConstructMessage(thread.Topic, "new-thread", "", conn.ID, payload)
	SendMessage(conn, response)
}

func replyHandler(conn *wsrooms.Conn, msg *wsrooms.Message) {
	var thread Thread
	var obj map[string]interface{}
	var dataobj []byte
	var reply Reply
	err := json.Unmarshal(msg.Payload, &reply)
	if err != nil {
		log.Println(err)
		return
	}
	reply.Unique.Generate()
	reply.FileInfo.Process()
	if obj = app.GetRow(reply.Topic, reply.Thread); err != nil {
		log.Println(err)
		return
	}
	dataobj, err = json.Marshal(obj)
	if err != nil {
		log.Println(err)
		return
	}
	err = json.Unmarshal(dataobj, &thread)
	if err != nil {
		log.Println(err)
		return
	}
	thread.Replies[reply.Unique.Hash] = reply
	for _, tag := range reply.Tagging {
		if tag == reply.Thread {
			thread.TaggedBy = append(thread.TaggedBy, reply.Unique.Hash)
		} else if taggedreply, ok := thread.Replies[tag]; ok {
			taggedreply.TaggedBy = append(taggedreply.TaggedBy, reply.Unique.Hash)
		}
	}
	thr, err := json.Marshal(&thread)
	if err != nil {
		log.Println(err)
		return
	}
	if err := app.InsertRow(reply.Topic, reply.Thread, string(thr)); err != nil {
		log.Println(err)
		return
	}
	payload, err := json.Marshal(&reply)
	if err != nil {
		log.Println(err)
		return
	}
	response := wsrooms.ConstructMessage(reply.Topic, "new-reply", "", conn.ID, payload)
	SendMessage(conn, response)
}

func main() {
	app = wsframe.NewApp("./config.json")
	wsrooms.Emitter.On("new-thread", threadHandler)
	wsrooms.Emitter.On("new-reply", replyHandler)
	app.Start()
}
