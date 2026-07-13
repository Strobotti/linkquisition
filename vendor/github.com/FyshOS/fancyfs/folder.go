package fancyfs

import (
	"errors"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
)

var ErrNoMetadata = errors.New("no metadata for requested folder")

type FancyFolder struct {
	BackgroundURI      fyne.URI
	BackgroundResource fyne.Resource
	BackgroundFill     canvas.ImageFill
}

func DetailsForFolder(dir fyne.URI) (*FancyFolder, error) {
	if dir == nil {
		return nil, nil
	}

	home, err := os.UserHomeDir()
	if err == nil {
		if dir.Path() == home {
			return &FancyFolder{
				BackgroundResource: theme.HomeIcon(),
				BackgroundFill:     canvas.ImageFillContain,
			}, nil
		}

		if isSubDir(dir.Path(), home) {
			var res fyne.Resource
			switch dir.Name() {
			case "Desktop":
				res = theme.DesktopIcon()
			case "Documents":
				res = theme.DocumentIcon()
			case "Downloads":
				res = theme.DownloadIcon()
			case "Music":
				res = theme.MediaMusicIcon()
			case "Pictures":
				res = theme.MediaPhotoIcon()
			case "Videos", "Movies":
				res = theme.MediaVideoIcon()
			}

			if res != nil {
				return &FancyFolder{
					BackgroundResource: res,
					BackgroundFill:     canvas.ImageFillContain,
				}, nil
			}
		}
	}

	err = ErrNoMetadata
	bg, err1 := checkBGImage(dir, ".background.png")
	if bg != nil {
		return bg, nil
	} else if err1 != ErrNoMetadata {
		err = err1
	}
	bg, err2 := checkBGImage(dir, ".background.jpg")
	if bg != nil {
		return bg, nil
	} else if err2 != ErrNoMetadata {
		err = err2
	}
	bg, err3 := checkBGImage(dir, ".background.jpeg")
	if bg != nil {
		return bg, nil
	} else if err3 != ErrNoMetadata {
		err = err3
	}
	bg, err4 := checkBGImage(dir, ".background.svg")
	if bg != nil {
		return bg, nil
	} else if err4 != ErrNoMetadata {
		err = err4
	}

	return nil, err
}

func checkBGImage(dir fyne.URI, name string) (*FancyFolder, error) {
	bgFile, err := storage.Child(dir, name)
	if err != nil {
		return nil, err
	}

	if yes, err := storage.Exists(bgFile); !yes || err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoMetadata
		}

		return nil, err
	}

	if filepath.Ext(name) == ".svg" {
		res, err := fyne.LoadResourceFromPath(bgFile.Path())
		if err == nil {
			// Fix the uniqueness of name
			res = fyne.NewStaticResource(bgFile.Path(), res.Content())

			return &FancyFolder{
				BackgroundResource: theme.NewThemedResource(res),
				BackgroundFill:     canvas.ImageFillContain,
				BackgroundURI:      bgFile,
			}, nil
		}
	}

	return &FancyFolder{
		BackgroundURI:  bgFile,
		BackgroundFill: canvas.ImageFillCover,
	}, nil
}

func isSubDir(dir, home string) bool {
	parent := filepath.Dir(dir)

	return parent == home
}
