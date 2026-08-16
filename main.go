// Package main is the entry point for the MoarChan web service. It coordinates
// image processing, REST endpoints, real-time Server-Sent Events (SSE), and routing.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	xdraw "golang.org/x/image/draw"

	"moarchan/frame"
)

// Global application container.
var app *frame.App

// tagRegex matches post reference links (e.g. ">>a1b2c3d4e").
var tagRegex = regexp.MustCompile(`&gt;&gt;([A-Za-z0-9]+)`)

const (
	// MaxImageDimension defines the maximum allowed pixel width or height to prevent decompression bombs.
	MaxImageDimension = 10000
	// MaxUploadSize defines the maximum multipart form memory limit (32MB).
	MaxUploadSize = 32 << 20
)

// Thread represents an original post (OP) and its associated replies.
type Thread struct {
	Hash           string   `json:"hash"`
	Topic          string   `json:"topic"`
	Name           string   `json:"name"`
	Subject        string   `json:"subject"`
	Options        string   `json:"options"`
	Comment        string   `json:"comment"`
	FileName       string   `json:"file_name"`
	FileMime       string   `json:"file_mime"`
	FileSize       string   `json:"file_size"`
	FileDimensions string   `json:"file_dimensions"`
	Timestamp      string   `json:"timestamp"`
	Replies        []Reply  `json:"replies"`
	TaggedBy       []string `json:"taggedBy"`
	Tagging        []string `json:"tagging"`
}

// Reply represents an individual response within a thread.
type Reply struct {
	Hash           string   `json:"hash"`
	Thread         string   `json:"thread"`
	Topic          string   `json:"topic"`
	Name           string   `json:"name"`
	Options        string   `json:"options"`
	Comment        string   `json:"comment"`
	FileName       string   `json:"file_name"`
	FileMime       string   `json:"file_mime"`
	FileSize       string   `json:"file_size"`
	FileDimensions string   `json:"file_dimensions"`
	Timestamp      string   `json:"timestamp"`
	TaggedBy       []string `json:"taggedBy"`
	Tagging        []string `json:"tagging"`
}

// FileDetails encapsulates metadata for an uploaded and processed image attachment.
type FileDetails struct {
	Name       string
	Path       string
	Mime       string
	Size       string
	Dimensions string
}

// sanitizeComment escapes HTML, formats greentext quotes, and transforms >>hash references into clickable tags.
func sanitizeComment(raw string) (string, []string) {
	escaped := html.EscapeString(raw)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")

	var tags []string
	lines := strings.Split(escaped, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		// Format post quotation links (>>hash)
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

// createThumbnail scales the source image to fit within maxW x maxH using
// bilinear interpolation (ApproxBiLinear) for significantly better quality
// than nearest-neighbor sampling, with minimal CPU overhead.
func createThumbnail(src image.Image, maxW, maxH int) image.Image {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxW && h <= maxH {
		return src
	}

	ratioW := float64(maxW) / float64(w)
	ratioH := float64(maxH) / float64(h)
	scale := ratioW
	if ratioH < ratioW {
		scale = ratioH
	}

	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// processMultipartUpload validates, strips metadata from, and generates thumbnails for uploaded images.
// It persists files via the Storage abstraction and cleans up orphaned files if thumbnail generation fails.
func processMultipartUpload(ctx context.Context, fileHeader *multipart.FileHeader, uniqueID string) (*FileDetails, error) {
	if fileHeader == nil {
		return &FileDetails{}, nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open uploaded file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}
	if !allowedExts[ext] {
		return nil, fmt.Errorf("file extension %q is not allowed", ext)
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("invalid or corrupted image binary: %w", err)
	}
	if config.Width > MaxImageDimension || config.Height > MaxImageDimension {
		return nil, fmt.Errorf("image dimensions (%dx%d) exceed maximum %dpx", config.Width, config.Height, MaxImageDimension)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("reset file pointer: %w", err)
	}

	baseName := filepath.Base(fileHeader.Filename)
	if baseName == "" || baseName == "." {
		baseName = "upload" + ext
	}
	safeName := fmt.Sprintf("%s_%s", uniqueID, baseName)
	thumbName := "thumb_" + safeName

	// Re-encode image binary to strip metadata and generate scaled thumbnail
	var imgData bytes.Buffer
	var thumbData bytes.Buffer

	switch format {
	case "jpeg":
		img, err := jpeg.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("decode jpeg binary: %w", err)
		}
		if err := jpeg.Encode(&imgData, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("re-encode jpeg: %w", err)
		}
		thumbImg := createThumbnail(img, 250, 250)
		if err := jpeg.Encode(&thumbData, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode jpeg thumbnail: %w", err)
		}
	case "png":
		img, err := png.Decode(file)
		if err != nil {
			return nil, fmt.Errorf("decode png binary: %w", err)
		}
		if err := png.Encode(&imgData, img); err != nil {
			return nil, fmt.Errorf("re-encode png: %w", err)
		}
		thumbImg := createThumbnail(img, 250, 250)
		if err := png.Encode(&thumbData, thumbImg); err != nil {
			return nil, fmt.Errorf("encode png thumbnail: %w", err)
		}
	case "gif":
		gifImg, err := gif.DecodeAll(file)
		if err != nil {
			return nil, fmt.Errorf("decode gif binary: %w", err)
		}
		if err := gif.EncodeAll(&imgData, gifImg); err != nil {
			return nil, fmt.Errorf("re-encode gif: %w", err)
		}
		if len(gifImg.Image) > 0 {
			thumbImg := createThumbnail(gifImg.Image[0], 250, 250)
			if err := jpeg.Encode(&thumbData, thumbImg, &jpeg.Options{Quality: 85}); err != nil {
				return nil, fmt.Errorf("encode gif thumbnail: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported image format %q", format)
	}

	// Persist via Storage abstraction (local disk or S3-compatible)
	if err := app.Storage.Save(ctx, safeName, bytes.NewReader(imgData.Bytes())); err != nil {
		return nil, fmt.Errorf("save image to storage: %w", err)
	}
	if thumbData.Len() > 0 {
		if err := app.Storage.Save(ctx, thumbName, bytes.NewReader(thumbData.Bytes())); err != nil {
			_ = app.Storage.Delete(ctx, safeName)
			return nil, fmt.Errorf("save thumbnail to storage: %w", err)
		}
	}

	return &FileDetails{
		Name:       safeName,
		Path:       app.Storage.PublicURL(safeName),
		Mime:       "image/" + format,
		Size:       fmt.Sprintf("%.1f", float64(imgData.Len())/1024.0),
		Dimensions: fmt.Sprintf("%dx%d", config.Width, config.Height),
	}, nil
}

// generateUnique generates a human-readable 4chan timestamp and a cryptographic 9-character hexadecimal hash.
func generateUnique() (string, string) {
	now := time.Now()
	timestamp := fmt.Sprintf(
		"%02d/%02d/%02d(%s)%02d:%02d:%02d",
		now.Month(),
		now.Day(),
		now.Year()%100,
		now.Weekday().String()[:3],
		now.Hour(),
		now.Minute(),
		now.Second(),
	)
	id, err := uuid.NewRandom()
	if err != nil {
		return timestamp, fmt.Sprintf("%x", sha256.Sum256([]byte(timestamp)))[:9]
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(timestamp+id.String())))[:9]
	return timestamp, hash
}

// hashPostPassword creates a bcrypt hash for post deletion passwords.
func hashPostPassword(pwd string) (string, error) {
	if pwd == "" {
		return "", nil
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	return string(h), err
}

// handleCreateThread processes HTTP POST requests to create a new thread with mandatory image attachment.
func handleCreateThread(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		http.Error(w, "Multipart form parse error or size limit exceeded", http.StatusBadRequest)
		return
	}

	topic := strings.TrimSpace(r.FormValue("topic"))
	rawName := strings.TrimSpace(r.FormValue("name"))
	rawSub := strings.TrimSpace(r.FormValue("subject"))
	rawOpt := strings.TrimSpace(r.FormValue("options"))
	rawCom := strings.TrimSpace(r.FormValue("comment"))
	rawPass := strings.TrimSpace(r.FormValue("password"))

	if topic == "" {
		http.Error(w, "Missing topic", http.StatusBadRequest)
		return
	}

	_, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Image file required to post a new thread", http.StatusBadRequest)
		return
	}

	timestamp, hash := generateUnique()
	fileDetails, err := processMultipartUpload(r.Context(), fileHeader, hash)
	if err != nil {
		log.Println("File upload error:", err)
		http.Error(w, fmt.Sprintf("File upload failed: %v", err), http.StatusBadRequest)
		return
	}

	comment, _ := sanitizeComment(rawCom)
	name := html.EscapeString(rawName)
	if name == "" {
		name = "Anonymous"
	}
	subject := html.EscapeString(rawSub)
	passHash, _ := hashPostPassword(rawPass)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	const insertQuery = `
INSERT INTO threads (hash, topic, name, subject, options, password_hash, comment, file_name, file_mime, file_size, file_dimensions, timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`
	_, err = app.Driver.ExecContext(ctx, insertQuery,
		hash, topic, name, subject, rawOpt, passHash, comment,
		fileDetails.Name, fileDetails.Mime, fileDetails.Size, fileDetails.Dimensions, timestamp,
	)
	if err != nil {
		log.Println("Insert thread error:", err)
		http.Error(w, "Database error creating thread", http.StatusInternalServerError)
		return
	}

	threadPayload := map[string]interface{}{
		"hash":            hash,
		"topic":           topic,
		"name":            name,
		"subject":         subject,
		"options":         rawOpt,
		"comment":         comment,
		"file_name":       fileDetails.Name,
		"file_mime":       fileDetails.Mime,
		"file_size":       fileDetails.Size,
		"file_dimensions": fileDetails.Dimensions,
		"timestamp":       timestamp,
		"replies":         []interface{}{},
		"taggedBy":        []string{},
		"tagging":         []string{},
	}

	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, "thread-item", threadPayload); err != nil {
		log.Printf("Execute template thread error: %v", err)
	} else {
		threadPayload["html"] = buf.String()
	}

	ssePayload, err := json.Marshal(threadPayload)
	if err != nil {
		http.Error(w, "Serialization error", http.StatusInternalServerError)
		return
	}

	app.Hub.Broadcast(topic, "new-thread", ssePayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(ssePayload)
}

// handleCreateReply processes HTTP POST requests to reply to an existing thread.
func handleCreateReply(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		http.Error(w, "Multipart form parse error or size limit exceeded", http.StatusBadRequest)
		return
	}

	topic := strings.TrimSpace(r.FormValue("topic"))
	threadHash := strings.TrimSpace(r.FormValue("thread"))
	rawName := strings.TrimSpace(r.FormValue("name"))
	rawOpt := strings.TrimSpace(r.FormValue("options"))
	rawCom := strings.TrimSpace(r.FormValue("comment"))
	rawPass := strings.TrimSpace(r.FormValue("password"))

	if topic == "" || threadHash == "" {
		http.Error(w, "Missing thread or topic parameter", http.StatusBadRequest)
		return
	}
	if rawCom == "" {
		http.Error(w, "Comment is required to post a reply", http.StatusBadRequest)
		return
	}

	var fileHeader *multipart.FileHeader
	file, header, err := r.FormFile("file")
	if err == nil {
		file.Close()
		fileHeader = header
	}

	timestamp, hash := generateUnique()
	fileDetails, err := processMultipartUpload(r.Context(), fileHeader, hash)
	if err != nil {
		log.Println("File upload error:", err)
		http.Error(w, fmt.Sprintf("File upload failed: %v", err), http.StatusBadRequest)
		return
	}

	comment, tags := sanitizeComment(rawCom)
	name := html.EscapeString(rawName)
	if name == "" {
		name = "Anonymous"
	}
	passHash, _ := hashPostPassword(rawPass)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := app.Driver.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Transaction error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM threads WHERE hash = $1 FOR UPDATE)`, threadHash).Scan(&exists); err != nil || !exists {
		http.Error(w, "Target thread not found", http.StatusNotFound)
		return
	}

	taggingJSON, _ := json.Marshal(tags)
	taggedByJSON, _ := json.Marshal([]string{})

	const insertPostQuery = `
INSERT INTO posts (hash, thread_hash, topic, name, options, password_hash, comment, file_name, file_mime, file_size, file_dimensions, timestamp, tagging, tagged_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`
	_, err = tx.ExecContext(ctx, insertPostQuery,
		hash, threadHash, topic, name, rawOpt, passHash, comment,
		fileDetails.Name, fileDetails.Mime, fileDetails.Size, fileDetails.Dimensions, timestamp,
		taggingJSON, taggedByJSON,
	)
	if err != nil {
		log.Println("Insert post error:", err)
		http.Error(w, "Database error creating reply", http.StatusInternalServerError)
		return
	}

	// Bump thread timestamp unless "sage" is in options
	if !strings.Contains(strings.ToLower(rawOpt), "sage") {
		_, _ = tx.ExecContext(ctx, `UPDATE threads SET bumped_at = CURRENT_TIMESTAMP WHERE hash = $1`, threadHash)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Commit transaction error", http.StatusInternalServerError)
		return
	}

	replyPayload := map[string]interface{}{
		"hash":            hash,
		"thread":          threadHash,
		"topic":           topic,
		"name":            name,
		"options":         rawOpt,
		"comment":         comment,
		"file_name":       fileDetails.Name,
		"file_mime":       fileDetails.Mime,
		"file_size":       fileDetails.Size,
		"file_dimensions": fileDetails.Dimensions,
		"timestamp":       timestamp,
		"taggedBy":        []string{},
		"tagging":         tags,
	}

	var buf bytes.Buffer
	if err := app.Templates.ExecuteTemplate(&buf, "reply-item", replyPayload); err != nil {
		log.Printf("Execute template reply error: %v", err)
	} else {
		replyPayload["html"] = buf.String()
	}

	ssePayload, err := json.Marshal(replyPayload)
	if err != nil {
		http.Error(w, "Serialization error", http.StatusInternalServerError)
		return
	}

	app.Hub.Broadcast(topic, "new-reply", ssePayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(ssePayload)
}

// handleDeletePost processes HTTP POST requests to delete posts or attachments using password verification.
func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimSpace(r.FormValue("hash"))
	password := strings.TrimSpace(r.FormValue("password"))
	fileOnly := r.FormValue("file_only") == "true" || r.FormValue("file_only") == "on"

	if hash == "" {
		http.Error(w, "Post or thread hash is required", http.StatusBadRequest)
		return
	}

	sessionVals, _ := app.GetSessionValues(r)
	isAdmin := sessionVals["privilege"] == "admin"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	info, err := DeletePostOrFile(ctx, app.Driver, hash, password, isAdmin, fileOnly)
	if err != nil {
		log.Println("Delete post error:", err)
		http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusForbidden)
		return
	}

	// Remove corresponding image files from storage
	for _, fn := range info.FileNames {
		if fn != "" {
			if err := app.Storage.Delete(ctx, fn); err != nil {
				log.Printf("Failed to delete file %q from storage: %v", fn, err)
			}
			if err := app.Storage.Delete(ctx, "thumb_"+fn); err != nil {
				log.Printf("Failed to delete thumbnail %q from storage: %v", fn, err)
			}
		}
	}

	eventPayload, _ := json.Marshal(map[string]interface{}{
		"hash":      info.Hash,
		"topic":     info.Topic,
		"file_only": fileOnly,
	})
	app.Hub.Broadcast(info.Topic, "delete-post", eventPayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"hash":      info.Hash,
		"file_only": fileOnly,
	})
}

// ResolveAndRenderEvent handles on-demand data hydration and HTML template rendering
// for compact SSE messages distributed via PostgreSQL pg_notify.
func ResolveAndRenderEvent(ctx context.Context, a *frame.App, topic, event string, rawData []byte) ([]byte, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal(rawData, &meta); err != nil {
		return rawData, nil
	}

	// Explicitly check if a database fetch was requested
	fetchRequired, _ := meta["fetch"].(bool)
	if !fetchRequired {
		return rawData, nil
	}

	hash, _ := meta["hash"].(string)
	if hash == "" {
		return rawData, nil
	}

	switch event {
	case "new-thread":
		threadData, err := GetSingleThread(ctx, a.Driver, topic, hash)
		if err != nil {
			return rawData, err
		}
		var buf bytes.Buffer
		if err := a.Templates.ExecuteTemplate(&buf, "thread-item", threadData); err == nil {
			threadData["html"] = buf.String()
		}
		return json.Marshal(threadData)
	case "new-reply":
		replyData, err := GetSinglePost(ctx, a.Driver, hash)
		if err != nil {
			return rawData, err
		}
		var buf bytes.Buffer
		if err := a.Templates.ExecuteTemplate(&buf, "reply-item", replyData); err == nil {
			replyData["html"] = buf.String()
		}
		return json.Marshal(replyData)
	}
	return rawData, nil
}

func main() {
	var err error
	app, err = frame.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// 1. Programmatic Route Declarations
	app.Route("/auth").Template("auth").Controller("auth")
	app.Route("/news").Template("news")
	app.Route("/blog").Template("blog")
	app.Route("/faq").Template("faq")
	app.Route("/rules").Template("rules")
	app.Route("/advertise").Template("advertise")
	app.Route("/press").Template("press")
	app.Route("/about").Template("about")
	app.Route("/feedback").Template("feedback")
	app.Route("/legal").Template("legal")
	app.Route("/contact").Template("contact")

	app.Route(`^/([A-Za-z0-9]+)/thread/([A-Za-z0-9]+)$`).
		Table("$1").
		Key("$2").
		Template("thread").
		Controller("service")

	app.Route(`^/([A-Za-z0-9]+)[\/]?$`).
		Table("$1").
		Template("topic").
		Controller("service")

	app.Route(`^/$`).
		Table("main").
		Template("main").
		Controller("main")

	// 2. Application Data Resolver Hook
	app.DataProvider = MoarchanDataProvider

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 3. Database Schema Initialization (versioned migrations)
	if err := InitMoarchanSchema(ctx, app); err != nil {
		log.Fatalf("Failed to initialize moarchan schema: %v", err)
	}

	// 4. Cluster-Safe SSE Event Resolver
	app.Hub.EventResolver = func(resolverCtx context.Context, topic, event string, rawData []byte) ([]byte, error) {
		return ResolveAndRenderEvent(resolverCtx, app, topic, event, rawData)
	}

	// 5. REST Endpoints (Go 1.22+ Native Method Routing)
	app.Router.HandleFunc("POST /api/threads", handleCreateThread)
	app.Router.HandleFunc("POST /api/replies", handleCreateReply)
	app.Router.HandleFunc("POST /api/posts/delete", handleDeletePost)

	if err := app.Start(); err != nil {
		log.Fatal("Application start error:", err)
	}
}
