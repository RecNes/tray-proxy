package main

// icongen builds a multi-size Windows .ico from assets/logo.png.
//
// Usage (repo root):
//
//	go run ./tools/icongen
//
// Input:  assets/logo.png (square, committed source of truth)
// Output: assets/trayproxy.ico (16,24,32,48,64,128,256 px PNG-compressed
// entries; committed so builds and the installer are deterministic).
//
// The exe file icon is embedded at link time from this .ico via
// akavel/rsrc in scripts/build-installer.ps1; the systray icon is
// embedded via go:embed in internal/tray/icon.go; the installer uses it
// via SetupIconFile in installer/trayproxy.iss.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

var iconSizes = []int{16, 24, 32, 48, 64, 128, 256}

const (
	srcRel = "assets/logo.png"
	dstRel = "assets/trayproxy.ico"
)

func main() {
	if err := run(srcRel, dstRel); err != nil {
		fmt.Fprintln(os.Stderr, "icongen:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", dstRel)
}

func run(srcPath, dstPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", srcPath, err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", srcPath, err)
	}
	if b := src.Bounds(); b.Dx() != b.Dy() {
		return fmt.Errorf("logo must be square, got %dx%d", b.Dx(), b.Dy())
	}
	ico, err := buildICO(src, iconSizes)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dstPath, ico, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}

// buildICO encodes each size as a PNG-compressed ICO entry (supported
// since Windows Vista; keeps alpha intact and avoids BMP writer code).
func buildICO(src image.Image, sizes []int) ([]byte, error) {
	type entry struct {
		size int
		png  []byte
	}
	entries := make([]entry, 0, len(sizes))
	base := toRGBA(src)
	for _, s := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, resizeBilinear(base, s, s)); err != nil {
			return nil, fmt.Errorf("encode %dx%d: %w", s, s, err)
		}
		entries = append(entries, entry{size: s, png: buf.Bytes()})
	}

	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	_ = binary.Write(&out, binary.LittleEndian, uint16(1)) // type: icon
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(entries)))

	offset := 6 + 16*len(entries)
	for _, e := range entries {
		w := byte(e.size)
		if e.size >= 256 {
			w = 0 // 0 means 256 in ICO headers
		}
		out.WriteByte(w)
		out.WriteByte(w)
		out.WriteByte(0) // color count
		out.WriteByte(0) // reserved
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))  // planes
		_ = binary.Write(&out, binary.LittleEndian, uint16(32)) // bit count
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(e.png)))
		_ = binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(e.png)
	}
	for _, e := range entries {
		out.Write(e.png)
	}
	return out.Bytes(), nil
}

func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// resizeBilinear scales src to w×h with bilinear interpolation.
func resizeBilinear(src *image.RGBA, w, h int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if sw == w && sh == h {
		copy(dst.Pix, src.Pix)
		return dst
	}
	for y := 0; y < h; y++ {
		gy := (float64(y)+0.5)*float64(sh)/float64(h) - 0.5
		y0 := int(gy)
		fy := gy - float64(y0)
		if y0 < 0 {
			y0, fy = 0, 0
		} else if y0 >= sh-1 {
			y0, fy = sh-1, 0
		}
		for x := 0; x < w; x++ {
			gx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			x0 := int(gx)
			fx := gx - float64(x0)
			if x0 < 0 {
				x0, fx = 0, 0
			} else if x0 >= sw-1 {
				x0, fx = sw-1, 0
			}
			var c [4]float64
			for ch := 0; ch < 4; ch++ {
				p00 := float64(src.Pix[(y0*sw+x0)*4+ch])
				p10 := float64(src.Pix[(y0*sw+x0+1)*4+ch])
				p01 := float64(src.Pix[((y0+1)*sw+x0)*4+ch])
				p11 := float64(src.Pix[((y0+1)*sw+x0+1)*4+ch])
				// Clamp the +1 samples at the edges (bilinear may read
				// one pixel past when x0 == sw-1 / y0 == sh-1).
				if x0+1 >= sw {
					p10, p11 = p00, p01
				}
				if y0+1 >= sh {
					p01, p11 = p00, p10
				}
				top := p00*(1-fx) + p10*fx
				bot := p01*(1-fx) + p11*fx
				c[ch] = top*(1-fy) + bot*fy
			}
			i := (y*w + x) * 4
			dst.Pix[i+0] = uint8(c[0] + 0.5)
			dst.Pix[i+1] = uint8(c[1] + 0.5)
			dst.Pix[i+2] = uint8(c[2] + 0.5)
			dst.Pix[i+3] = uint8(c[3] + 0.5)
		}
	}
	return dst
}
