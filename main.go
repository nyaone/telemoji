package main

import (
	"encoding/json"
	"flag"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"telemoji/types"
	"time"
)

var (
	flagConfig string
	flagOutDir string
	flagHost   string
	flagHelp   bool
)

var (
	packShortIDRegex = regexp.MustCompile("https://t.me/addemoji/(.+)")
)

func init() {
	flag.StringVar(&flagConfig, "config", "config.json", "Config file path")
	flag.StringVar(&flagOutDir, "outdir", "packs", "Emoji pack save directory")
	flag.StringVar(&flagHost, "host", "nya.one", "Instance for emoji meta")
	flag.BoolVar(&flagHelp, "help", false, "Print help message")
}

func main() {
	// Parse command line args
	flag.Parse()

	targetPacks := flag.Args()
	if flagHelp || len(targetPacks) == 0 {
		// Requires help
		fmt.Println("Usage: telemoji [args] emojiPackLink")
		flag.PrintDefaults()
		return
	}

	var validPacks []string
	for _, packFullLink := range targetPacks {
		matches := packShortIDRegex.FindStringSubmatch(packFullLink)
		if len(matches) > 1 {
			validPacks = append(validPacks, matches[1])
		} else {
			log.Printf("Invalid pack: %s", packFullLink)
		}
	}
	if len(validPacks) == 0 {
		log.Fatalf("No valid packs")
		return
	}

	log.Printf("Downloading valid packs: %v", validPacks)

	// Prepare out dir
	err := os.Mkdir(flagOutDir, 0750)
	if err != nil {
		log.Fatalf("Failed to prepare output directory")
	}

	// Parse config
	var cfg Config
	configFileBytes, err := os.ReadFile(flagConfig)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	err = json.Unmarshal(configFileBytes, &cfg)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Start tg bot
	bot, err := tgbotapi.NewBotAPI(cfg.TGBotToken)
	if err != nil {
		log.Fatalf("Failed to initialize tg bot: %v", err)
	}

	bot.Debug = true

	log.Printf("Authorize on account %s", bot.Self.UserName)

	// Start download pack
	for _, packShortID := range validPacks {
		stickerSet, err := bot.GetStickerSet(tgbotapi.GetStickerSetConfig{
			Name: packShortID,
		})
		if err != nil {
			log.Printf("Failed to get pack %s, skip", packShortID)
			continue
		}

		log.Printf("Downloading pack %s...", stickerSet.Title)

		// Prepare output dir
		outDir := path.Join(flagOutDir, packShortID)
		err = os.Mkdir(outDir, 0750)
		if err != nil {
			log.Printf("Failed to prepare directory for pack %s, skip", packShortID)
			continue
		}

		// Prepare pack meta
		packEmojis := make([]types.Emoji, len(stickerSet.Stickers))

		for i, sticker := range stickerSet.Stickers {
			// Prepare output file
			emojiID := fmt.Sprintf("%s_%d", packShortID, i+1)

			// Set meta info
			packEmojis[i].Downloaded = false
			packEmojis[i].Emoji.Name = emojiID
			packEmojis[i].Emoji.Category = stickerSet.Title
			if sticker.Emoji != "" {
				packEmojis[i].Emoji.Aliases = append(packEmojis[i].Emoji.Aliases, sticker.Emoji)
			}

			// Download file
			stickerFileLink, err := bot.GetFileDirectURL(sticker.FileID)
			if err != nil {
				log.Printf("Failed to get file %s with error: %v", sticker.FileID, err)
				continue
			}
			res, err := (&http.Client{}).Get(stickerFileLink)
			if err != nil {
				log.Printf("Failed to create request %s with error: %v", stickerFileLink, err)
				continue
			}

			// Check mimetype
			var fileExt string
			contentType := res.Header.Get("Content-Type")
			switch contentType {
			//case "image/jpeg":
			//	fileExt = "jpg"
			//case "image/png":
			//	fileExt = "png"
			//case "image/svg+xml":
			//	fileExt = "svg"
			//case "image/webp":
			//	fileExt = "webp"
			// is always application/octet-stream // TODO: auto-detect
			default:
				// No match, use png as fallback
				fileExt = "png"
			}

			filename := fmt.Sprintf("%s.%s", emojiID, fileExt)
			outFile, err := os.Create(path.Join(outDir, filename))
			if err != nil {
				log.Printf("Failed to prepare output file %s with error: %v", filename, err)
				continue
			}

			_, err = io.Copy(outFile, res.Body)
			res.Body.Close()
			outFile.Close()
			if err != nil {
				log.Printf("Failed to create request %s with error: %v", stickerFileLink, err)
				continue
			}

			log.Printf("File download successfully: %s", filename)
			packEmojis[i].FileName = filename
			packEmojis[i].Downloaded = true
		}

		// Save pack meta
		packMeta := types.Meta{
			MetaVersion: 2,
			Host:        flagHost,
			ExportedAt:  time.Now(),
			Emojis:      packEmojis,
		}

		packMetaBytes, err := json.MarshalIndent(&packMeta, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal pack meta %v with error: %v", packMeta, err)
			continue
		}

		packMetaFilename := path.Join(outDir, "meta.json")
		err = os.WriteFile(packMetaFilename, packMetaBytes, 0644)
		if err != nil {
			log.Printf("Failed to save pack meta json with error: %v", err)
			continue
		}
	}

	log.Printf("All packs downloaded, enjoy.")

}
