package main

import (
	"encoding/base64"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
)

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func encodeImageToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(buf, data)
	return string(buf), nil
}

func cropAndSave(img image.Image, box [4]int, dest string) error {
	rect := image.Rect(box[0], box[1], box[2], box[3])
	if !rect.In(img.Bounds()) {
		rect = rect.Intersect(img.Bounds())
	}
	if rect.Empty() {
		return nil
	}
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, rect.Min, draw.Src)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	opts := jpeg.Options{Quality: 95}
	return jpeg.Encode(file, cropped, &opts)
}

