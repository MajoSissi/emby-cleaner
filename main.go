package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	embyAPIPath = "/emby/Users/%s/Items"
)

type Config struct {
	Emby struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"emby"`
	Cleanup struct {
		WatchedDaysAgo     int      `yaml:"watched_days_ago"`
		KeepLatestEpisodes int      `yaml:"keep_latest_episodes"`
		LibraryNames       []string `yaml:"library_names"`
		TagFilters         []string `yaml:"tag_filters"`
		ProtectTags        []string `yaml:"protect_tags"`
		ProtectFavorites   bool     `yaml:"protect_favorites"`
		DryRun             bool     `yaml:"dry_run"`
		RemoveEmptyFolders bool     `yaml:"remove_empty_folders"`
	} `yaml:"cleanup"`
}

type EmbyItem struct {
	ID                string   `json:"Id"`
	Name              string   `json:"Name"`
	Type              string   `json:"Type"`
	Path              string   `json:"Path"`
	UserData          UserData `json:"UserData"`
	SeriesName        string   `json:"SeriesName"`
	ParentIndexNumber int      `json:"ParentIndexNumber"`
	IndexNumber       int      `json:"IndexNumber"`
	SeriesID          string   `json:"SeriesId"`
	SeasonID          string   `json:"SeasonId"`
	Tags              []string `json:"Tags"`
}

type UserData struct {
	LastPlayedDate   string  `json:"LastPlayedDate"`
	PlaybackPosition float64 `json:"PlaybackPosition"`
	Played           bool    `json:"Played"`
	IsFavorite       bool    `json:"IsFavorite"`
}

type EmbyResponse struct {
	Items            []EmbyItem `json:"Items"`
	TotalRecordCount int        `json:"TotalRecordCount"`
}

type EmbyClient struct {
	baseURL   string
	authToken string
	userID    string
	client    *http.Client
}

func NewEmbyClient(baseURL string) *EmbyClient {
	return &EmbyClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *EmbyClient) Authenticate(username, password string) error {
	apiURL := fmt.Sprintf("%s/emby/Users/authenticatebyname", c.baseURL)

	authData := map[string]string{
		"Username": username,
		"Pw":       password,
	}

	jsonData, err := json.Marshal(authData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Client", "Emby Auto Cleaner")
	req.Header.Set("X-Emby-Client-Version", "1.0")
	req.Header.Set("X-Emby-Device-Name", "EmbyAutoCleaner")
	req.Header.Set("X-Emby-Device-Id", "emby-auto-cleaner")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s, body: %s", resp.Status, string(body))
	}

	var result struct {
		AccessToken string `json:"AccessToken"`
		User        struct {
			ID string `json:"Id"`
		} `json:"User"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	c.authToken = result.AccessToken
	c.userID = result.User.ID
	return nil
}

func (c *EmbyClient) GetUserID() string {
	return c.userID
}

func (c *EmbyClient) GetItems(userID, itemType string, filters map[string]string) ([]EmbyItem, error) {
	apiURL := fmt.Sprintf("%s%s", c.baseURL, fmt.Sprintf(embyAPIPath, userID))

	params := url.Values{}
	params.Add("Recursive", "true")
	params.Add("IncludeItemTypes", itemType)

	for k, v := range filters {
		params.Add(k, v)
	}

	apiURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Emby-Token", c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result EmbyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (c *EmbyClient) GetLibraries() (map[string]string, error) {
	apiURL := fmt.Sprintf("%s/emby/Library/RefreshStatus", c.baseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Emby-Token", c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Name string `json:"Name"`
			ID   string `json:"ItemId"`
		} `json:"Items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	libraries := make(map[string]string)
	for _, item := range result.Items {
		libraries[item.Name] = item.ID
	}

	return libraries, nil
}

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func shouldDelete(item EmbyItem, config *Config, watchedCutoff time.Time) bool {
	if item.UserData.IsFavorite && config.Cleanup.ProtectFavorites {
		return false
	}

	if item.Type != "Episode" {
		return false
	}

	if item.Path == "" {
		return false
	}

	if len(config.Cleanup.ProtectTags) > 0 {
		for _, protectTag := range config.Cleanup.ProtectTags {
			for _, itemTag := range item.Tags {
				if strings.EqualFold(itemTag, protectTag) {
					return false
				}
			}
		}
	}

	if !item.UserData.Played {
		return false
	}

	if item.UserData.LastPlayedDate == "" {
		return false
	}

	lastPlayed, err := time.Parse(time.RFC3339, item.UserData.LastPlayedDate)
	if err != nil {
		return false
	}

	if lastPlayed.After(watchedCutoff) {
		return false
	}

	return true
}

func removeEmptyFolders(basePath string) error {
	var folders []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			folders = append(folders, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	for i := len(folders) - 1; i >= 0; i-- {
		folder := folders[i]
		if folder == basePath {
			continue
		}

		isEmpty, err := isDirEmpty(folder)
		if err != nil {
			continue
		}

		if isEmpty {
			fmt.Printf("删除空文件夹: %s\n", folder)
			os.Remove(folder)
		}
	}

	return nil
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func main() {
	configPath := "emby-auto-cleaner.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	client := NewEmbyClient(config.Emby.URL)

	err = client.Authenticate(config.Emby.Username, config.Emby.Password)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		os.Exit(1)
	}

	userID := client.GetUserID()
	fmt.Printf("登录成功，用户ID: %s\n", userID)

	filters := make(map[string]string)

	libraryIDs := make(map[string]bool)
	if len(config.Cleanup.LibraryNames) > 0 {
		libraries, err := client.GetLibraries()
		if err != nil {
			fmt.Printf("获取媒体库列表失败: %v\n", err)
		} else {
			for _, libName := range config.Cleanup.LibraryNames {
				if libID, exists := libraries[libName]; exists {
					libraryIDs[libID] = true
					fmt.Printf("添加媒体库: %s\n", libName)
				}
			}

			if len(libraryIDs) > 0 {
				var ids []string
				for id := range libraryIDs {
					ids = append(ids, id)
				}
				filters["ParentId"] = strings.Join(ids, ",")
			}
		}
	}

	if len(config.Cleanup.TagFilters) > 0 {
		filters["Tags"] = strings.Join(config.Cleanup.TagFilters, ",")
	}

	items, err := client.GetItems(userID, "Episode", filters)
	if err != nil {
		fmt.Printf("获取剧集列表失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("找到 %d 个剧集\n", len(items))

	watchedCutoff := time.Now().AddDate(0, 0, -config.Cleanup.WatchedDaysAgo)
	fmt.Printf("观看截止时间: %s (watched_days_ago=%d)\n", watchedCutoff.Format("2006-01-02 15:04:05"), config.Cleanup.WatchedDaysAgo)

	var itemsToDelete []EmbyItem
	watchedCount := 0
	for _, item := range items {
		if item.UserData.Played {
			watchedCount++
		}
		if shouldDelete(item, config, watchedCutoff) {
			itemsToDelete = append(itemsToDelete, item)
		}
	}

	fmt.Printf("已观看的剧集数: %d\n", watchedCount)
	if watchedCount > 0 {
		for _, item := range items {
			if item.UserData.Played {
				fmt.Printf("  - %s (S%02dE%02d) Played: %v, LastPlayed: %s, Path: %s\n",
					item.SeriesName, item.ParentIndexNumber, item.IndexNumber,
					item.UserData.Played, item.UserData.LastPlayedDate, item.Path)
			}
		}
	}
	fmt.Printf("准备删除 %d 个已观看剧集\n", len(itemsToDelete))

	seriesEpisodes := make(map[string][]EmbyItem)
	for _, item := range itemsToDelete {
		key := item.SeriesID
		seriesEpisodes[key] = append(seriesEpisodes[key], item)
	}

	for _, episodes := range seriesEpisodes {
		if len(episodes) <= config.Cleanup.KeepLatestEpisodes {
			continue
		}

		sortEpisodesByIndexNumber(episodes)

		keepCount := config.Cleanup.KeepLatestEpisodes
		deleteCount := len(episodes) - keepCount

		for i := 0; i < deleteCount; i++ {
			if config.Cleanup.DryRun {
				fmt.Printf("[DRY RUN] 将删除: %s - S%02dE%02d - %s\n",
					episodes[i].SeriesName,
					episodes[i].ParentIndexNumber,
					episodes[i].IndexNumber,
					episodes[i].Path)
			} else {
				fmt.Printf("删除: %s - S%02dE%02d - %s\n",
					episodes[i].SeriesName,
					episodes[i].ParentIndexNumber,
					episodes[i].IndexNumber,
					episodes[i].Path)

				if err := os.Remove(episodes[i].Path); err != nil {
					fmt.Printf("删除失败: %v\n", err)
				}
			}
		}
	}

	if config.Cleanup.RemoveEmptyFolders && !config.Cleanup.DryRun {
		fmt.Println("\n清理空文件夹...")
		for libID := range libraryIDs {
			libPath, err := client.GetLibraryPath(libID)
			if err != nil {
				fmt.Printf("获取媒体库路径失败: %v\n", err)
				continue
			}
			if libPath != "" {
				removeEmptyFolders(libPath)
			}
		}
	}

	if config.Cleanup.DryRun {
		fmt.Println("\n[DRY RUN] 这是模拟运行，没有实际删除文件")
	} else {
		fmt.Println("\n清理完成")
	}
}

func sortEpisodesByIndexNumber(episodes []EmbyItem) {
	for i := 0; i < len(episodes)-1; i++ {
		for j := i + 1; j < len(episodes); j++ {
			if episodes[i].ParentIndexNumber > episodes[j].ParentIndexNumber ||
				(episodes[i].ParentIndexNumber == episodes[j].ParentIndexNumber &&
					episodes[i].IndexNumber > episodes[j].IndexNumber) {
				episodes[i], episodes[j] = episodes[j], episodes[i]
			}
		}
	}
}
