package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
    "strconv"
    "encoding/json"
    "time"
    "sort"
    "bytes"
    "mime/multipart"
    htmlmd "github.com/JohannesKaufmann/html-to-markdown"

    "blog/cmd/statix_cmd/mdtostatix"
    "blog/cmd/statix_cmd/statixtoclean"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func bold(s string) string   { return colorBold + s + colorReset }
func green(s string) string  { return colorGreen + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func cyan(s string) string   { return colorCyan + s + colorReset }

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func nicknameFromFile(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

type Config struct {
	URL   string
	Token string
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".statix_config")
}

func saveConfig(urlStr, token string) error {
	content := fmt.Sprintf("%s\n%s\n", urlStr, token)
	return os.WriteFile(configPath(), []byte(content), 0600)
}

func loadConfig() (*Config, error) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid config file")
	}
	return &Config{
		URL:   lines[0],
		Token: lines[1],
	}, nil
}

func publish(title string,
	subjectID string,
	isPublic string,
	filePath string) (int64, error) {

	cfg, err := loadConfig()
	if err != nil {
		return 0, err
	}

	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	content := string(rawContent)

	if strings.HasSuffix(strings.ToLower(filePath), ".md") {
		htmlContent, err := mdtostatix.MarkdownToStatixHTML(content)
		if err != nil {
			return 0, fmt.Errorf("markdown conversion failed: %w", err)
		}
		content = htmlContent
	}

	data := url.Values{}
	data.Set("title", title)
	data.Set("subject_id", subjectID)
	data.Set("is_public", isPublic)
	data.Set("html", content)

	req, err := http.NewRequest("POST", cfg.URL+"/admin/new", strings.NewReader(data.Encode()))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	idStr := strings.TrimSpace(string(body))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse article ID: %s", idStr)
	}

	return id, nil
}

func editArticle(id, title, subjectID, isPublic, filePath string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(rawContent)

	if strings.HasSuffix(strings.ToLower(filePath), ".md") {

		htmlContent, err := mdtostatix.MarkdownToStatixHTML(content)
		if err != nil {
			return fmt.Errorf("markdown conversion failed: %w", err)
		}

		content = htmlContent
	}

	data := url.Values{}
	data.Set("title", title)
	data.Set("subject_id", subjectID)
	data.Set("is_public", isPublic)
	data.Set("html", content)

	endpoint := fmt.Sprintf("%s/admin/articles/%s", cfg.URL, id)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	return nil
}

func listNicknames() error {
	store, err := loadNicknames()
	if err != nil {
		return err
	}

	if len(store) == 0 {
		fmt.Println("No nicknames defined.")
		return nil
	}

	// Extract keys
	names := make([]string, 0, len(store))
	for name := range store {
		names = append(names, name)
	}

	// Sort alphabetically
	sort.Strings(names)

	// Print in order
	for _, name := range names {
		meta := store[name]

		status := yellow("unpublished")
		if meta.ArticleID != 0 {
			status = green(fmt.Sprintf("id=%d", meta.ArticleID))
		}

		fmt.Printf(
			"%s  %s  %s  %s\n",
			bold(name),
			cyan(meta.Title),
			fmt.Sprintf("(subject=%d, public=%t)", meta.SubjectID, meta.IsPublic),
			status,
		)
	}

	return nil
}

func listSubjects() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", cfg.URL+"/admin/api/subjects", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

    resp, err := httpClient.Do(req)
	
    if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	fmt.Print(string(body))
	return nil
}

func listArticles() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", cfg.URL+"/admin/api/articles", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	fmt.Print(string(body))
	return nil
}

type ArticleMeta struct {
	Title     string `json:"title"`
	SubjectID int64  `json:"subject_id"`
	IsPublic  bool   `json:"is_public"`
	ArticleID int64  `json:"article_id,omitempty"`
}

type NicknameStore map[string]ArticleMeta

func nicknameFilePath() string {
	return ".statix_articles.json"
}

func loadNicknames() (NicknameStore, error) {
	path := nicknameFilePath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NicknameStore{}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var store NicknameStore
	if err := json.Unmarshal(b, &store); err != nil {
		return nil, err
	}

	return store, nil
}

func getNickname(name string) (ArticleMeta, error) {
	store, err := loadNicknames()
	if err != nil {
		return ArticleMeta{}, err
	}

	meta, ok := store[name]
	if !ok {
		return ArticleMeta{}, fmt.Errorf("nickname not found")
	}

	return meta, nil
}

func saveNicknames(store NicknameStore) error {
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(nicknameFilePath(), b, 0644)
}

func createNickname(name string, 
                    title string, 
                    subjectID int64, 
                    isPublic bool) error {
	store, err := loadNicknames()
	if err != nil {
		return err
	}

	if _, exists := store[name]; exists {
		return fmt.Errorf("nickname already exists")
	}

	store[name] = ArticleMeta{
		Title:     title,
		SubjectID: subjectID,
		IsPublic:  isPublic,
	}

	return saveNicknames(store)
}

func setArticleID(name string, id int64) error {
	store, err := loadNicknames()
	if err != nil {
		return err
	}

	meta, ok := store[name]
	if !ok {
		return fmt.Errorf("nickname not found")
	}

	meta.ArticleID = id
	store[name] = meta

	return saveNicknames(store)
}

func removeNickname(name string) error {
	store, err := loadNicknames()
	if err != nil {
		return err
	}

	if _, exists := store[name]; !exists {
		return fmt.Errorf("nickname not found")
	}

	delete(store, name)

	return saveNicknames(store)
}

func deleteRemoteArticle(id int64) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/delete/%d", cfg.URL, id)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	return nil
}

func renameNickname(oldName, newName string) error {
	store, err := loadNicknames()
	if err != nil {
		return err
	}

	meta, exists := store[oldName]
	if !exists {
		return fmt.Errorf("nickname not found")
	}

	if _, already := store[newName]; already {
		return fmt.Errorf("target nickname already exists")
	}

	// Move entry
	store[newName] = meta
	delete(store, oldName)

	return saveNicknames(store)
}

func importNickname(articleID int64, nickname string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/api/articles/%d", cfg.URL, articleID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	parts := strings.Split(strings.TrimSpace(string(body)), "\t")
	if len(parts) != 3 {
		return fmt.Errorf("invalid server response: %s", string(body))
	}

	title := parts[0]

	subjectID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid subject_id from server")
	}

	isPublic, err := strconv.ParseBool(parts[2])
	if err != nil {
		return fmt.Errorf("invalid is_public from server")
	}

	store, err := loadNicknames()
	if err != nil {
		return err
	}

	if _, exists := store[nickname]; exists {
		return fmt.Errorf("nickname already exists")
	}

	store[nickname] = ArticleMeta{
		Title:     title,
		SubjectID: subjectID,
		IsPublic:  isPublic,
		ArticleID: articleID,
	}

	return saveNicknames(store)
}

func importContent(articleID int64, nickname string, asMarkdown bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/api/articles-content/%d", cfg.URL, articleID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	content := string(body)

	if asMarkdown {

        cleanHTML, tables, err := statixtoclean.StripStatixWrappers(content)
        if err != nil {
            return err
        }

        converter := htmlmd.NewConverter("", true, nil)
		md, err := converter.ConvertString(cleanHTML)
		if err != nil {
			return err
		}
      
        for i, table := range tables {
        
        	placeholder := fmt.Sprintf("STATIXTABLETOKEN%d", i)
        
        	md = strings.ReplaceAll(
        		md,
        		placeholder,
        		"\n\n"+table+"\n\n",
        	)
        }

        md = strings.ReplaceAll(md, "* * *", "---")
        md = strings.ReplaceAll(md, `\[x\]`, `[x]`)
        md = strings.ReplaceAll(md, `\[ \]`, `[ ]`)
		
        content = md
	}

	ext := ".html"
	if asMarkdown {
		ext = ".md"
	}

	filename := nickname + ext

	return os.WriteFile(filename, []byte(content), 0644)
}

func uploadFiles(paths []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

    for _, path := range paths {
    	file, err := os.Open(path)
    	if err != nil {
    		return err
    	}
    
    	safeName := filepath.Base(path)
    
    	part, err := writer.CreateFormFile("files", safeName)
    	if err != nil {
    		file.Close()
    		return err
    	}
    
    	if _, err := io.Copy(part, file); err != nil {
    		file.Close()
    		return err
    	}
    
    	file.Close()
    }

	if err := writer.Close(); err != nil {
		return err
	}

	url := cfg.URL + "/admin/files"

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(bodyBytes))
	}

	return nil
}

func deleteFileRemote(name string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/admin/files/delete/%s", cfg.URL, name)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	return nil
}

func listFilesRemote() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", cfg.URL+"/admin/api/files", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Print(string(body))
	return nil
}

func usage() {
	fmt.Println("stx - Statix Publishing CLI")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  set-credentials --url URL --password TOKEN")
	fmt.Println("  publish --file FILE [NAME]")
	fmt.Println("  nickname create --title TITLE --subject_id ID --is_public true|false NAME")
    fmt.Println("  nickname import ARTICLE_ID NAME")
    fmt.Println("  nickname import-content [--markdown] ARTICLE_ID NAME")
	fmt.Println("  nickname remove [--sync] NAME")
    fmt.Println("  nickname list")
    fmt.Println("  nickname rename OLD_NAME NEW_NAME")
    fmt.Println("  file upload FILE...")
    fmt.Println("  file delete FILE")
    fmt.Println("  file list")
	fmt.Println("  articles")
	fmt.Println("  subjects")
	fmt.Println()
}

func main() {

	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {

	// ---------------- set-credentials ----------------

	case "set-credentials":
		cmd := flag.NewFlagSet("set-credentials", flag.ExitOnError)
		urlStr := cmd.String("url", "", "Base URL")
		password := cmd.String("password", "", "Token")
		cmd.Parse(os.Args[2:])

		if *urlStr == "" || *password == "" {
			fmt.Println("Missing --url or --password")
			return
		}

		if err := saveConfig(*urlStr, *password); err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println("Credentials saved.")

	// ---------------- publish ----------------

    case "publish":
    	cmd := flag.NewFlagSet("publish", flag.ExitOnError)
    	file := cmd.String("file", "", "HTML/MD file path")
    	cmd.Parse(os.Args[2:])
    
    	if *file == "" {
    		fmt.Println("Missing --file")
    		return
    	}
    
    	var name string
    
    	if cmd.NArg() >= 1 {
    		name = cmd.Arg(0)
    	} else {
    		name = nicknameFromFile(*file)
    		fmt.Println("Auto-detected nickname:", name)
    	}
	
        if strings.HasSuffix(strings.ToLower(*file), ".md") {
        	fmt.Println("Detected Markdown → converting to Statix HTML")
        }

		meta, err := getNickname(name)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		subIDStr := strconv.FormatInt(meta.SubjectID, 10)
		publicStr := strconv.FormatBool(meta.IsPublic)

		if meta.ArticleID != 0 {
			err = editArticle(
				strconv.FormatInt(meta.ArticleID, 10),
				meta.Title,
				subIDStr,
				publicStr,
				*file,
			)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println("Article updated.")
			return
		}

		newID, err := publish(meta.Title, subIDStr, publicStr, *file)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if err := setArticleID(name, newID); err != nil {
			fmt.Println("Published but failed to update nickname:", err)
			return
		}

		fmt.Println("Article published and nickname updated.")

	// ---------------- nickname ----------------

	case "nickname":
		if len(os.Args) < 3 {
			fmt.Println("Usage: stx nickname [create|import|import-content|remove|rename|list]")
			return
		}

		switch os.Args[2] {

		case "create":
			cmd := flag.NewFlagSet("nickname create", flag.ExitOnError)
			title := cmd.String("title", "", "Article title")
			subjectID := cmd.String("subject_id", "", "Subject ID")
			isPublic := cmd.String("is_public", "true", "Visibility")
			cmd.Parse(os.Args[3:])

			if cmd.NArg() < 1 {
				fmt.Println("Usage: stx nickname create NAME --title ... --subject_id ...")
				return
			}

			name := cmd.Arg(0)

			if *title == "" || *subjectID == "" {
				fmt.Println("Missing --title or --subject_id")
				return
			}

			subID, err := strconv.ParseInt(*subjectID, 10, 64)
			if err != nil {
				fmt.Println("Invalid subject_id")
				return
			}

			publicBool, err := strconv.ParseBool(*isPublic)
			if err != nil {
				fmt.Println("Invalid is_public value")
				return
			}

			if err := createNickname(name, *title, subID, publicBool); err != nil {
				fmt.Println("Error:", err)
				return
			}

			fmt.Println("Nickname created.")

        case "import":
        	cmd := flag.NewFlagSet("nickname import", flag.ExitOnError)
        	cmd.Parse(os.Args[3:])
        
        	if cmd.NArg() < 2 {
        		fmt.Println("Usage: stx nickname import ARTICLE_ID NAME")
        		return
        	}
        
        	idStr := cmd.Arg(0)
        	name := cmd.Arg(1)
        
        	id, err := strconv.ParseInt(idStr, 10, 64)
        	if err != nil {
        		fmt.Println("Invalid article ID")
        		return
        	}
        
        	if err := importNickname(id, name); err != nil {
        		fmt.Println("Error:", err)
        		return
        	}
        
        	fmt.Println("Nickname imported successfully.")

        case "remove":
        	cmd := flag.NewFlagSet("nickname remove", flag.ExitOnError)
        	syncFlag := cmd.Bool("sync", false, "Delete remote article too")
        	cmd.Parse(os.Args[3:])
        
        	if cmd.NArg() < 1 {
        		fmt.Println("Usage: stx nickname remove NAME [--sync]")
        		return
        	}
        
        	name := cmd.Arg(0)
        
        	meta, err := getNickname(name)
        	if err != nil {
        		fmt.Println("Error:", err)
        		return
        	}
        
        	if *syncFlag && meta.ArticleID != 0 {
        		if err := deleteRemoteArticle(meta.ArticleID); err != nil {
        			fmt.Println("Remote deletion failed:", err)
        			return
        		}
        		fmt.Println("Remote article deleted.")
        	}
        
        	if err := removeNickname(name); err != nil {
        		fmt.Println("Error:", err)
        		return
        	}
        
            fmt.Println("Nickname removed.")

        case "import-content":
        	cmd := flag.NewFlagSet("nickname import-content", flag.ExitOnError)
            asMarkdown := cmd.Bool("markdown", false, "Convert to Markdown")
        	cmd.Parse(os.Args[3:])
        
        	if cmd.NArg() < 2 {
                fmt.Println("Usage: stx nickname import-content ARTICLE_ID NAME [--markdown]")
        		return
        	}
        
        	idStr := cmd.Arg(0)
        	name := cmd.Arg(1)
        
        	id, err := strconv.ParseInt(idStr, 10, 64)
        	if err != nil {
        		fmt.Println("Invalid article ID")
        		return
        	}
        
        	if err := importContent(id, name, *asMarkdown); err != nil {
        		fmt.Println("Error:", err)
        		return
        	}

        case "list":
            if err := listNicknames(); err != nil {
                fmt.Println("Error:", err)
            }

        case "rename":
        	cmd := flag.NewFlagSet("nickname rename", flag.ExitOnError)
        	cmd.Parse(os.Args[3:])
        
        	if cmd.NArg() < 2 {
        		fmt.Println("Usage: stx nickname rename OLD_NAME NEW_NAME")
        		return
        	}
        
        	oldName := cmd.Arg(0)
        	newName := cmd.Arg(1)
        
        	if err := renameNickname(oldName, newName); err != nil {
        		fmt.Println("Error:", err)
        		return
        	}
        
        	fmt.Println("Nickname renamed.")

		default:
			fmt.Println("Unknown nickname command.")
		}

	// ---------------- articles ----------------

	case "articles":
		if err := listArticles(); err != nil {
			fmt.Println("Error:", err)
		}

	// ---------------- subjects ----------------

	case "subjects":
		if err := listSubjects(); err != nil {
			fmt.Println("Error:", err)
		}

    case "file":
    	if len(os.Args) < 3 {
    		fmt.Println("Usage: stx file [upload|delete|list]")
    		return
    	}
    
    	switch os.Args[2] {
    
    	case "upload":
    		cmd := flag.NewFlagSet("file upload", flag.ExitOnError)
    		cmd.Parse(os.Args[3:])
    
    		if cmd.NArg() < 1 {
    			fmt.Println("Usage: stx file upload FILE...")
    			return
    		}
    
    		if err := uploadFiles(cmd.Args()); err != nil {
    			fmt.Println("Error:", err)
    			return
    		}
    
    		fmt.Println("Files uploaded successfully.")
    
        case "delete":
		    cmd := flag.NewFlagSet("file delete", flag.ExitOnError)
		    cmd.Parse(os.Args[3:])

		    if cmd.NArg() < 1 {
		    	fmt.Println("Usage: stx file delete NAME")
		    	return
		    }

		    name := cmd.Arg(0)

		    if err := deleteFileRemote(name); err != nil {
		    	fmt.Println("Error:", err)
		    	return
		    }

		    fmt.Println("File deleted successfully.")
    
        case "list":
        	if err := listFilesRemote(); err != nil {
        		fmt.Println("Error:", err)
        	}

        default:
    		fmt.Println("Unknown file command.")
    	}

	default:
		usage()
	}
}




