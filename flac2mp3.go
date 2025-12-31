package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func flacToMp3ViaFFmpeg(inputPath, outputPath string) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("file not exists: %s", inputPath)
	}
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-ab", "320k", "-ac", "2", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trans error: %v\nFFmpeg output: %s", err, string(output))
	}
	return nil
}

func copyFile(src, dest string) error {
	dstFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file '%s' error: %w", dest, err)
	}
	defer func() {
		_ = dstFile.Close()
	}()
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open file '%s' error: %w", src, err)
	}
	defer func() {
		_ = srcFile.Close()
	}()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copy file '%s' to '%s' error: %w", src, dest, err)
	}
	return nil
}

func main() {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.Fatal("required ffmpeg not found in $PATH")
	}

	flacDir := flag.String("flac", "", "source dir containing .flac files")
	mp3Dir := flag.String("mp3", "", "destination dir of output .mp3 files")
	concurrency := flag.Uint("concurrency", 1, "trans concurrency")
	flag.Parse()
	if *flacDir == "" || *mp3Dir == "" {
		flag.Usage()
		os.Exit(1)
	}

	ch := make(chan struct{}, *concurrency)
	err := filepath.Walk(*flacDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || err != nil {
			return err
		}

		outputPath := filepath.Join(*mp3Dir, strings.Replace(filepath.Base(path), ".flac", ".mp3", -1))
		if tmpInfo, tmpErr := os.Stat(outputPath); tmpErr == nil && tmpInfo.Size() > 0 {
			log.Printf("skip already exists file: %s\n", path)
			return nil
		}

		if filepath.Ext(path) == ".mp3" {
			if err = copyFile(path, outputPath); err != nil {
				return err
			}
		}
		if filepath.Ext(path) != ".flac" {
			return nil
		}

		ch <- struct{}{}
		go func() {
			defer func() { <-ch }()
			if err = flacToMp3ViaFFmpeg(path, outputPath); err != nil {
				log.Printf("trans %s failed：%s\n", path, err)
			}
			log.Printf("trans success: %s -> %s\n", path, outputPath)
		}()
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
