// main.go
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/joncody/roomer"
	"github.com/vincent-petithory/dataurl"
	"moarchan/frame"
)

var app *frame.App

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

func (f *FileInfo) Process() error {
	if f.File == "" {
		return nil
	}

	// 1. Decode Data URL
	fdata, err := dataurl.DecodeString(f.File)
	if err != nil {
		log.Println("DataURL decode error:", err)
		return err
	}

	// 2. Sanitize filename to prevent path traversal
	safeName := filepath.Base(f.Name)
	if safeName == "" || safeName == "." {
		safeName = fmt.Sprintf("upload_%d.jpg", time.Now().UnixNano())
	}
	f.Name = safeName
	f.Path = fmt.Sprintf("./static/images/uploads/%s", safeName)

	// 3. Ensure upload directory exists
	if err := os.MkdirAll("./static/images/uploads", 0755); err != nil {
		log.Println("MkdirAll error:", err)
		return err
	}

	// 4. Write image file to disk
	if err := os.WriteFile(f.Path, fdata.Data, 0644); err != nil {
		log.Println("WriteFile error:", err)
		return err
	}

	// 5. Decode image dimensions directly from binary header bytes
	config, _, err := image.DecodeConfig(bytes.NewReader(fdata.Data))
	if err != nil {
		log.Println("Image DecodeConfig warning:", err)
		f.Dimensions = "N/A"
	} else {
		f.Dimensions = fmt.Sprintf("%dx%d", config.Width, config.Height)
	}

	// 6. Calculate accurate file size in KB
	f.Size = fmt.Sprintf("%.1f", float64(len(fdata.Data))/1024.0)

	// 7. Clear raw base64 payload from memory
	f.File = ""
	return nil
}

func (u *Unique) Generate() {
	now := time.Now()
	u.Timestamp = fmt.Sprintf(
		"%d/%d/%d(%s)%02d:%02d:%02d",
		now.Month(),
		now.Day(),
		now.Year(),
		now.Weekday().String()[:3],
		now.Hour(),
		now.Minute(),
		now.Second(),
	)
	id, err := uuid.NewRandom()
	if err != nil {
		log.Println(err)
		return
	}
	u.Uuid = id.String()
	u.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(u.Timestamp+u.Uuid)))[:9]
}

func SendMessage(conn *roomer.Conn, msg *roomer.Message) {
	conn.SendToRoom(msg.Room, msg.Event, msg.Payload)
	conn.TrySend(msg.Bytes())
}

func threadHandler(conn *roomer.Conn, msg *roomer.Message) error {
	var thread Thread
	err := json.Unmarshal(msg.Payload, &thread)
	if err != nil {
		log.Println(err)
		return err
	}
	if thread.Topic == "" {
		return fmt.Errorf("invalid thread payload: missing topic")
	}
	thread.Unique.Generate()
	if err := thread.FileInfo.Process(); err != nil {
		log.Println("File processing warning:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.InsertRow(ctx, thread.Topic, thread.Unique.Hash, &thread); err != nil {
		log.Println(err)
		return err
	}
	payload, err := json.Marshal(&thread)
	if err != nil {
		log.Println(err)
		return err
	}
	response := roomer.NewMessage(thread.Topic, "new-thread", "", conn.ID, payload)
	SendMessage(conn, response)
	return nil
}

func replyHandler(conn *roomer.Conn, msg *roomer.Message) error {
	var reply Reply
	err := json.Unmarshal(msg.Payload, &reply)
	if err != nil {
		log.Println(err)
		return err
	}
	if reply.Thread == "" || reply.Topic == "" {
		return fmt.Errorf("invalid reply payload: missing thread or topic")
	}
	reply.Unique.Generate()
	if err := reply.FileInfo.Process(); err != nil {
		log.Println("File processing warning:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var thread Thread
	if err := app.GetRowStruct(ctx, reply.Topic, reply.Thread, &thread); err != nil {
		log.Println(err)
		return err
	}

	if thread.Replies == nil {
		thread.Replies = make(map[string]Reply)
	}
	thread.Replies[reply.Unique.Hash] = reply

	for _, tag := range reply.Tagging {
		if tag == reply.Thread {
			thread.TaggedBy = append(thread.TaggedBy, reply.Unique.Hash)
		} else if taggedReply, exists := thread.Replies[tag]; exists {
			taggedReply.TaggedBy = append(taggedReply.TaggedBy, reply.Unique.Hash)
			thread.Replies[tag] = taggedReply
		}
	}

	if err := app.InsertRow(ctx, reply.Topic, reply.Thread, &thread); err != nil {
		log.Println(err)
		return err
	}

	payload, err := json.Marshal(&reply)
	if err != nil {
		log.Println(err)
		return err
	}
	response := roomer.NewMessage(reply.Topic, "new-reply", "", conn.ID, payload)
	SendMessage(conn, response)
	return nil
}

func main() {
	var err error
	app, err = frame.NewApp("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := roomer.RegisterHandler("new-thread", threadHandler); err != nil {
		log.Fatal("Failed to register handler:", err)
	}
	if err := roomer.RegisterHandler("new-reply", replyHandler); err != nil {
		log.Fatal("Failed to register handler:", err)
	}
	if err := app.Start(); err != nil {
		log.Fatal("Application start error:", err)
	}
}
