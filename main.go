package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joncody/roomer"
	"github.com/vincent-petithory/dataurl"
	"moarchan/frame"
)

var app *frame.App
var tagRegex = regexp.MustCompile(`&gt;&gt;([A-Za-z0-9]+)`)

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

func sanitizeComment(raw string) (string, []string) {
	escaped := html.EscapeString(raw)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")

	var tags []string
	lines := strings.Split(escaped, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		formattedLine := tagRegex.ReplaceAllStringFunc(line, func(match string) string {
			postHash := match[8:] // strip &gt;&gt;
			tags = append(tags, postHash)
			return fmt.Sprintf(`<span class="post-tag blue-text-link" data-tag="%s">%s</span>`, postHash, match)
		})

		// Format greentext lines (>quote)
		if strings.HasPrefix(formattedLine, "&gt;") && !strings.HasPrefix(formattedLine, "&gt;&gt;") {
			formattedLine = fmt.Sprintf(`<span class="post-quote">%s</span>`, formattedLine)
		}

		formattedLines = append(formattedLines, formattedLine)
	}

	return strings.Join(formattedLines, "<br />"), tags
}

func (f *FileInfo) Process(uniqueID string) error {
	if f.File == "" {
		return nil
	}

	// 1. Decode Data URL
	fdata, err := dataurl.DecodeString(f.File)
	if err != nil {
		log.Println("DataURL decode error:", err)
		return fmt.Errorf("invalid base64 image data: %w", err)
	}

	// 2. Validate file extension whitelist to prevent Stored XSS
	ext := strings.ToLower(filepath.Ext(f.Name))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}
	if !allowedExts[ext] {
		return fmt.Errorf("file extension %q is not allowed", ext)
	}

	// 3. Decode binary header to verify image integrity and format
	config, format, err := image.DecodeConfig(bytes.NewReader(fdata.Data))
	if err != nil {
		log.Println("Image DecodeConfig error:", err)
		return fmt.Errorf("invalid or corrupted image binary: %w", err)
	}

	f.Dimensions = fmt.Sprintf("%dx%d", config.Width, config.Height)
	f.Mime = "image/" + format

	// 4. Sanitize filename and prevent collisions
	baseName := filepath.Base(f.Name)
	if baseName == "" || baseName == "." {
		baseName = "upload" + ext
	}
	safeName := fmt.Sprintf("%s_%s", uniqueID, baseName)
	f.Name = safeName
	f.Path = fmt.Sprintf("./static/images/uploads/%s", safeName)

	// 5. Ensure upload directory exists
	if err := os.MkdirAll("./static/images/uploads", 0755); err != nil {
		log.Println("MkdirAll error:", err)
		return err
	}

	// 6. Write image file to disk with strict permissions
	if err := os.WriteFile(f.Path, fdata.Data, 0600); err != nil {
		log.Println("WriteFile error:", err)
		return err
	}

	// 7. Calculate file size in KB
	f.Size = fmt.Sprintf("%.1f", float64(len(fdata.Data))/1024.0)

	// 8. Clear raw payload from memory
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
		log.Println("Unmarshal error:", err)
		return err
	}
	if thread.Topic == "" {
		return fmt.Errorf("invalid thread payload: missing topic")
	}
	if thread.FileInfo.File == "" {
		return fmt.Errorf("image file required to post a new thread")
	}

	// Validate topic table name safety
	if !frame.IsValidTableName(thread.Topic) || frame.IsSystemTable(thread.Topic) {
		return fmt.Errorf("invalid or restricted topic table: %q", thread.Topic)
	}

	thread.Unique.Generate()
	thread.Name = html.EscapeString(thread.Name)
	thread.Subject = html.EscapeString(thread.Subject)
	thread.Comment, _ = sanitizeComment(thread.Comment)

	if err := thread.FileInfo.Process(thread.Unique.Hash); err != nil {
		log.Println("File processing error:", err)
		return fmt.Errorf("file upload failed: %w", err)
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
		log.Println("Unmarshal error:", err)
		return err
	}
	if reply.Thread == "" || reply.Topic == "" {
		return fmt.Errorf("invalid reply payload: missing thread or topic")
	}

	// Validate topic table name safety before raw SQL transaction
	if !frame.IsValidTableName(reply.Topic) || frame.IsSystemTable(reply.Topic) {
		return fmt.Errorf("invalid or restricted topic table: %q", reply.Topic)
	}

	reply.Unique.Generate()
	reply.Name = html.EscapeString(reply.Name)

	var tags []string
	reply.Comment, tags = sanitizeComment(reply.Comment)
	reply.Tagging = tags

	if err := reply.FileInfo.Process(reply.Unique.Hash); err != nil {
		log.Println("File processing error:", err)
		return fmt.Errorf("file upload failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure target table exists safely via memory-cached frame helper
	if err := app.EnsureTable(ctx, reply.Topic); err != nil {
		return fmt.Errorf("ensure topic table %q: %w", reply.Topic, err)
	}

	// Use Transaction + Row Lock (FOR UPDATE) to prevent race condition on concurrent replies
	tx, err := app.Driver.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var value []byte
	query := fmt.Sprintf(`SELECT value FROM "%s" WHERE key = $1 FOR UPDATE`, reply.Topic)
	if err := tx.QueryRowContext(ctx, query, reply.Thread).Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("thread %q not found in topic %q", reply.Thread, reply.Topic)
		}
		return err
	}

	var thread Thread
	if err := json.Unmarshal(value, &thread); err != nil {
		return fmt.Errorf("unmarshal thread: %w", err)
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

	updatedData, err := json.Marshal(&thread)
	if err != nil {
		return fmt.Errorf("marshal updated thread: %w", err)
	}

	upsertQuery := fmt.Sprintf(`
		INSERT INTO "%s" (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		reply.Topic)

	if _, err := tx.ExecContext(ctx, upsertQuery, reply.Thread, updatedData); err != nil {
		return fmt.Errorf("upsert thread: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
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
