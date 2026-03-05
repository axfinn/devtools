package handlers

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxImageCache = 1000 // 最大缓存图片数
)

var (
	cacheDir       string
	cacheDirOnce   sync.Once
	cacheDirInit   error
	bgImages       []BackgroundImage
	bgImagesOnce  sync.Once
	bgImagesInit  sync.Mutex
	rng           = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// BackgroundImage represents a background image
type BackgroundImage struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Source    string `json:"source"` // "external", "cached", "local"
	Filename  string `json:"filename,omitempty"`
	CachedURL string `json:"cached_url,omitempty"`
}

func getCacheDir() string {
	cacheDirOnce.Do(func() {
		// 使用环境变量或默认路径（项目目录下的 data/backgrounds）
		bgPath := os.Getenv("BACKGROUND_CACHE_PATH")
		if bgPath == "" {
			bgPath = "./data/backgrounds"
		}

		// 确保目录存在
		if err := os.MkdirAll(bgPath, 0755); err != nil {
			cacheDirInit = err
			log.Printf("创建背景图缓存目录失败: %v", err)
		} else {
			cacheDir = bgPath
			log.Printf("背景图缓存目录: %s", bgPath)
		}
	})
	return cacheDir
}

// initBgImages 初始化背景图列表
func initBgImages() {
	bgImagesOnce.Do(func() {
		bgImagesInit.Lock()
		defer bgImagesInit.Unlock()

		images := getBgImageList()
		bgImages = images
	})
}

// InitBackgroundImages 公开的初始化函数，用于启动时调用
func InitBackgroundImages() {
	// 确保缓存目录存在
	getCacheDir()
	// 初始化图片列表
	initBgImages()
}

// getBgImageList 获取背景图列表
func getBgImageList() []BackgroundImage {
	var images []BackgroundImage
	cachePath := getCacheDir()

	// 本地缓存的图片优先
	if cachePath != "" {
		if entries, err := os.ReadDir(cachePath); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && isImageFile(entry.Name()) {
					id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
					images = append(images, BackgroundImage{
						ID:        id,
						URL:       fmt.Sprintf("/api/bg/cached/%s", entry.Name()),
						Source:    "cached",
						Filename:  entry.Name(),
						CachedURL: fmt.Sprintf("/api/bg/cached/%s", entry.Name()),
					})
				}
			}
		}
	}

	// 如果缓存不足1000张，添加外部图片
	if len(images) < maxImageCache {
		externalImages := []string{
			"https://picsum.photos/1920/1080?random=1",
			"https://picsum.photos/1920/1080?random=2",
			"https://picsum.photos/1920/1080?random=3",
			"https://picsum.photos/1920/1080?random=4",
			"https://picsum.photos/1920/1080?random=5",
			"https://picsum.photos/1920/1080?random=6",
			"https://picsum.photos/1920/1080?random=7",
			"https://picsum.photos/1920/1080?random=8",
			"https://picsum.photos/1920/1080?random=9",
			"https://picsum.photos/1920/1080?random=10",
			"https://picsum.photos/1920/1080?random=11",
			"https://picsum.photos/1920/1080?random=12",
			"https://picsum.photos/1920/1080?random=13",
			"https://picsum.photos/1920/1080?random=14",
			"https://picsum.photos/1920/1080?random=15",
			"https://picsum.photos/1920/1080?random=16",
			"https://picsum.photos/1920/1080?random=17",
			"https://picsum.photos/1920/1080?random=18",
			"https://picsum.photos/1920/1080?random=19",
			"https://picsum.photos/1920/1080?random=20",
		}

		for _, url := range externalImages {
			id := strings.TrimPrefix(url, "https://picsum.photos/1920/1080?random=")
			img := BackgroundImage{
				ID:     id,
				URL:    url,
				Source: "external",
			}

			// 检查是否有本地缓存
			if cachePath != "" {
				cachedFile := filepath.Join(cachePath, fmt.Sprintf("picsum-%s.jpg", id))
				if _, err := os.Stat(cachedFile); err == nil {
					img.CachedURL = fmt.Sprintf("/api/bg/cached/picsum-%s.jpg", id)
					img.Source = "cached"
				}
			}

			images = append(images, img)
		}
	}

	return images
}

// GetBackgroundImages returns list of background images
func GetBackgroundImages(c *gin.Context) {
	initBgImages()

	// 如果指定了刷新参数，重新加载
	if c.Query("refresh") == "true" {
		bgImagesInit.Lock()
		bgImages = getBgImageList()
		bgImagesInit.Unlock()
	}

	c.JSON(http.StatusOK, gin.H{
		"images":      bgImages,
		"total":       len(bgImages),
		"cached":      countCachedImages(),
		"max_cache":  maxImageCache,
	})
}

func countCachedImages() int {
	cachePath := getCacheDir()
	if cachePath == "" {
		return 0
	}
	count := 0
	if entries, err := os.ReadDir(cachePath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && isImageFile(entry.Name()) {
				count++
			}
		}
	}
	return count
}

// CacheBackgroundImages 缓存背景图到本地
func CacheBackgroundImages(c *gin.Context) {
	cachePath := getCacheDir()
	if cachePath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "缓存目录不可用"})
		return
	}

	// 获取需要缓存的图片列表
	externalImages := []string{
		"https://picsum.photos/1920/1080?random=1",
		"https://picsum.photos/1920/1080?random=2",
		"https://picsum.photos/1920/1080?random=3",
		"https://picsum.photos/1920/1080?random=4",
		"https://picsum.photos/1920/1080?random=5",
		"https://picsum.photos/1920/1080?random=6",
		"https://picsum.photos/1920/1080?random=7",
		"https://picsum.photos/1920/1080?random=8",
		"https://picsum.photos/1920/1080?random=9",
		"https://picsum.photos/1920/1080?random=10",
		"https://picsum.photos/1920/1080?random=11",
		"https://picsum.photos/1920/1080?random=12",
		"https://picsum.photos/1920/1080?random=13",
		"https://picsum.photos/1920/1080?random=14",
		"https://picsum.photos/1920/1080?random=15",
		"https://picsum.photos/1920/1080?random=16",
		"https://picsum.photos/1920/1080?random=17",
		"https://picsum.photos/1920/1080?random=18",
		"https://picsum.photos/1920/1080?random=19",
		"https://picsum.photos/1920/1080?random=20",
	}

	cached := 0
	failed := 0

	// 创建支持代理的 HTTP 客户端
	proxyURL := os.Getenv("HTTP_PROXY")
	var transport *http.Transport
	if proxyURL != "" {
		if proxy, err := http.ProxyFromEnvironment(&http.Request{URL: &url.URL{}}); err == nil {
			transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
		}
	}
	httpClient := &http.Client{Transport: transport}

	for _, imgURL := range externalImages {
		id := strings.TrimPrefix(imgURL, "https://picsum.photos/1920/1080?random=")
		filename := fmt.Sprintf("picsum-%s.jpg", id)
		cachedPath := filepath.Join(cachePath, filename)

		// 如果已存在则跳过
		if _, err := os.Stat(cachedPath); err == nil {
			continue
		}

		// 检查是否超过最大缓存数
		if countCachedImages() >= maxImageCache {
			break
		}

		// 下载图片
		resp, err := httpClient.Get(imgURL)
		if err != nil {
			log.Printf("下载图片失败 %s: %v", imgURL, err)
			failed++
			continue
		}
		defer resp.Body.Close()

		// 检查状态码
		if resp.StatusCode != http.StatusOK {
			log.Printf("下载图片失败 %s: status %d", imgURL, resp.StatusCode)
			failed++
			continue
		}

		// 保存到本地
		file, err := os.Create(cachedPath)
		if err != nil {
			log.Printf("创建文件失败 %s: %v", cachedPath, err)
			failed++
			continue
		}
		defer file.Close()

		if _, err := io.Copy(file, resp.Body); err != nil {
			log.Printf("保存图片失败 %s: %v", cachedPath, err)
			failed++
			continue
		}

		cached++
		log.Printf("已缓存图片: %s", filename)
	}

	// 刷新图片列表
	bgImagesInit.Lock()
	bgImages = getBgImageList()
	bgImagesInit.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message":    fmt.Sprintf("缓存完成，成功: %d, 失败: %d，当前缓存: %d/%d", cached, failed, countCachedImages(), maxImageCache),
		"cached":     cached,
		"failed":     failed,
		"total":      countCachedImages(),
		"max_cache":  maxImageCache,
	})
}

// ReplaceRandomImages 随机替换缓存中的图片
func ReplaceRandomImages(c *gin.Context) {
	cachePath := getCacheDir()
	if cachePath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "缓存目录不可用"})
		return
	}

	// 获取要替换的数量
	count := 10 // 默认替换10张
	if n := c.Query("count"); n != "" {
		fmt.Sscanf(n, "%d", &count)
	}

	// 确保不超过最大缓存数
	if count > maxImageCache {
		count = maxImageCache
	}

	// 创建支持代理的 HTTP 客户端
	proxyURL := os.Getenv("HTTP_PROXY")
	var transport *http.Transport
	if proxyURL != "" {
		if proxy, err := http.ProxyFromEnvironment(&http.Request{URL: &url.URL{}}); err == nil {
			transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
		}
	}
	httpClient := &http.Client{Transport: transport}

	replaced := 0
	failed := 0

	for i := 0; i < count; i++ {
		// 随机生成一个新的 random ID
		randomID := rng.Intn(10000) + 1
		imgURL := fmt.Sprintf("https://picsum.photos/1920/1080?random=%d", randomID)
		id := fmt.Sprintf("picsum-%d", randomID)
		filename := fmt.Sprintf("%s.jpg", id)
		cachedPath := filepath.Join(cachePath, filename)

		// 如果已存在则跳过
		if _, err := os.Stat(cachedPath); err == nil {
			continue
		}

		// 下载图片
		resp, err := httpClient.Get(imgURL)
		if err != nil {
			log.Printf("下载图片失败 %s: %v", imgURL, err)
			failed++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			failed++
			continue
		}

		// 保存到本地
		file, err := os.Create(cachedPath)
		if err != nil {
			failed++
			continue
		}
		defer file.Close()

		if _, err := io.Copy(file, resp.Body); err != nil {
			failed++
			continue
		}

		replaced++
	}

	// 刷新图片列表
	bgImagesInit.Lock()
	bgImages = getBgImageList()
	bgImagesInit.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("替换完成，替换: %d, 失败: %d", replaced, failed),
		"replaced":  replaced,
		"failed":    failed,
		"total":     countCachedImages(),
	})
}

// ServeCachedBackground serves cached background images
func ServeCachedBackground(c *gin.Context) {
	filename := c.Param("filename")
	cachePath := getCacheDir()

	if cachePath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "缓存目录不可用"})
		return
	}

	filePath := filepath.Join(cachePath, filename)
	c.File(filePath)
}

// GetRandomBackground 返回随机背景图
func GetRandomBackground(c *gin.Context) {
	initBgImages()

	if len(bgImages) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "没有可用的背景图"})
		return
	}

	// 优先返回本地缓存的图片
	var cachedImages []BackgroundImage
	for _, img := range bgImages {
		if img.Source == "cached" || img.Source == "local" {
			cachedImages = append(cachedImages, img)
		}
	}

	var img BackgroundImage
	if len(cachedImages) > 0 {
		img = cachedImages[rng.Intn(len(cachedImages))]
	} else {
		img = bgImages[rng.Intn(len(bgImages))]
	}

	// 重定向到图片 URL
	c.Redirect(http.StatusFound, img.URL)
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
