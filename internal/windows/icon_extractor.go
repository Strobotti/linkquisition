//go:build windows

package windows

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	iconCacheDir  = "icon-cache"
	iconCacheSize = 256 // Preferred icon size in pixels
)

var (
	shell32                = windows.NewLazySystemDLL("shell32.dll")
	user32                 = windows.NewLazySystemDLL("user32.dll")
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procExtractIconExW     = shell32.NewProc("ExtractIconExW")
	procDestroyIcon        = user32.NewProc("DestroyIcon")
	procGetIconInfo        = user32.NewProc("GetIconInfo")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
	procGetObject          = gdi32.NewProc("GetObjectW")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

// ICONINFO matches the Win32 ICONINFO structure.
type iconInfo struct {
	FIcon    uint32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// BITMAP matches the Win32 BITMAP structure.
type bitmap struct {
	BmType       int32
	BmWidth      int32
	BmHeight     int32
	BmWidthBytes int32
	BmPlanes     uint16
	BmBitsPixel  uint16
	BmBits       uintptr
}

// BITMAPINFOHEADER matches the Win32 BITMAPINFOHEADER structure.
type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// extractIconFromExe extracts the largest icon from an exe file and returns it as PNG bytes.
func extractIconFromExe(exePath string) ([]byte, error) {
	pathPtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Extract the first large icon from the exe
	var hIconLarge uintptr
	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0, // icon index
		uintptr(unsafe.Pointer(&hIconLarge)),
		0, // no small icon needed
		1, // extract 1 icon
	)
	if ret == 0 || hIconLarge == 0 {
		return nil, fmt.Errorf("no icon found in %s", exePath)
	}
	defer procDestroyIcon.Call(hIconLarge) //nolint:errcheck

	// Convert HICON to PNG
	return hIconToPNG(hIconLarge)
}

// hIconToPNG converts a Win32 HICON handle to PNG-encoded bytes.
func hIconToPNG(hIcon uintptr) ([]byte, error) {
	// Get icon info to access the bitmaps
	var info iconInfo
	ret, _, _ := procGetIconInfo.Call(hIcon, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return nil, fmt.Errorf("GetIconInfo failed")
	}
	defer func() {
		if info.HbmColor != 0 {
			procDeleteObject.Call(info.HbmColor) //nolint:errcheck
		}
		if info.HbmMask != 0 {
			procDeleteObject.Call(info.HbmMask) //nolint:errcheck
		}
	}()

	if info.HbmColor == 0 {
		return nil, fmt.Errorf("icon has no color bitmap")
	}

	// Get bitmap dimensions
	var bm bitmap
	procGetObject.Call(info.HbmColor, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))) //nolint:errcheck

	width := int(bm.BmWidth)
	height := int(bm.BmHeight)
	if width == 0 || height == 0 {
		return nil, fmt.Errorf("icon has zero dimensions")
	}

	// Create a device context and extract the pixel data
	hdc, _, _ := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdc) //nolint:errcheck

	// Prepare BITMAPINFOHEADER for 32-bit BGRA pixel data
	bih := bitmapInfoHeader{
		BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		BiWidth:       int32(width),
		BiHeight:      -int32(height), // negative = top-down
		BiPlanes:      1,
		BiBitCount:    32, //nolint:mnd
		BiCompression: 0,  // BI_RGB
	}

	// Allocate buffer for pixel data (4 bytes per pixel: BGRA)
	pixelDataSize := width * height * 4 //nolint:mnd
	pixelData := make([]byte, pixelDataSize)

	procSelectObject.Call(hdc, info.HbmColor) //nolint:errcheck
	ret, _, _ = procGetDIBits.Call(
		hdc,
		info.HbmColor,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixelData[0])),
		uintptr(unsafe.Pointer(&bih)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	// Convert BGRA pixel data to Go image.NRGBA
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			offset := (y*width + x) * 4 //nolint:mnd
			b := pixelData[offset]
			g := pixelData[offset+1]
			r := pixelData[offset+2]
			a := pixelData[offset+3]
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("PNG encode failed: %w", err)
	}

	return buf.Bytes(), nil
}

// iconCachePath returns the file path for a cached icon PNG.
func iconCachePath(exePath string) string {
	configDir, err := os.UserCacheDir()
	if err != nil {
		configDir = os.TempDir()
	}

	// Hash the exe path to create a stable filename
	hash := sha256.Sum256([]byte(strings.ToLower(exePath)))
	filename := hex.EncodeToString(hash[:8]) + ".png"

	return filepath.Join(configDir, "linkquisition", iconCacheDir, filename)
}

// getIconCached returns the icon for an exe path, using a local PNG cache.
// If the cached file exists, it's returned directly. Otherwise the icon is
// extracted and written to the cache.
func getIconCached(exePath string) ([]byte, error) {
	cachePath := iconCachePath(exePath)

	// Try cache first
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	// Extract the icon
	data, err := extractIconFromExe(exePath)
	if err != nil {
		return nil, err
	}

	// Write to cache (best-effort, don't fail if cache write fails)
	if mkErr := os.MkdirAll(filepath.Dir(cachePath), 0o755); mkErr == nil {
		_ = os.WriteFile(cachePath, data, 0o644)
	}

	return data, nil
}
