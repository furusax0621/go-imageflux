package imageflux

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
)

// Image is an image hosted on ImageFlux.
type Image struct {
	Path   string
	Proxy  *Proxy
	Config *Config
}

// Config is configure of image.
type Config struct {
	// Scaling Parameters.
	Width          int
	Height         int
	DisableEnlarge bool
	AspectMode     AspectMode
	Origin         Origin
	Background     color.Color

	// TODO: Overlay Parameters.

	// Output Parameters.
	Format              Format
	Quality             int
	DisableOptimization bool
}

// AspectMode is aspect mode.
type AspectMode int

const (
	// AspectModeDefault is the default value of aspect mode.
	AspectModeDefault AspectMode = iota

	// AspectModeScale holds the the aspect ratio of the input image,
	// and scales to fit in the specified size.
	AspectModeScale

	// AspectModeForceScale ignores the aspect ratio of the input image.
	AspectModeForceScale

	// AspectModeCrop holds the the aspect ratio of the input image,
	// and crops the image.
	AspectModeCrop

	// AspectModePad holds the the aspect ratio of the input image,
	// and fills the unfilled portion with the specified background color.
	AspectModePad
)

// Origin is the origin.
type Origin int

const (
	OriginDefault Origin = iota
	OriginTopLeft
	OriginTopCenter
	OriginTopRight
	OriginMiddleLeft
	OriginMiddleCenter
	OriginMiddleRight
	OriginBottomLeft
	OriginBottomCenter
	OriginBottomRight
)

// Format is the format of the output image.
type Format string

const (
	// FormatAuto encodes the image by the same format with the input image.
	FormatAuto Format = "auto"

	// FormatJPEG encodes the image as a JPEG.
	FormatJPEG Format = "jpg"

	// FormatPNG encodes the image as a PNG.
	FormatPNG Format = "png"

	// FormatGIF encodes the image as a GIF.
	FormatGIF Format = "gif"

	// FormatWebPFromJPEG encodes the image as a WebP.
	// The input image should be a JPEG.
	FormatWebPFromJPEG Format = "webp:jpeg"

	// FormatWebPFromPNG encodes the image as a WebP.
	// The input image should be a PNG.
	FormatWebPFromPNG Format = "webp:png"
)

func (c *Config) String() string {
	if c == nil {
		return ""
	}

	var buf []byte
	if c.Width != 0 {
		buf = append(buf, 'w', '=')
		buf = strconv.AppendInt(buf, int64(c.Width), 10)
		buf = append(buf, ',')
	}
	if c.Height != 0 {
		buf = append(buf, 'h', '=')
		buf = strconv.AppendInt(buf, int64(c.Height), 10)
		buf = append(buf, ',')
	}
	if c.DisableEnlarge {
		buf = append(buf, 'u', '=', '0', ',')
	}
	if c.AspectMode != AspectModeDefault {
		buf = append(buf, 'a', '=')
		buf = strconv.AppendInt(buf, int64(c.AspectMode-1), 10)
		buf = append(buf, ',')
	}
	if c.Origin != OriginDefault {
		buf = append(buf, 'g', '=')
		buf = strconv.AppendInt(buf, int64(c.Origin), 10)
		buf = append(buf, ',')
	}
	if c.Background != nil {
		r, g, b, a := c.Background.RGBA()
		if a == 0xffff {
			c := fmt.Sprintf("b=%02x%02x%02x,", r>>8, g>>8, b>>8)
			buf = append(buf, c...)
		} else if a == 0 {
			buf = append(buf, "b=000000"...)
		} else {
			r = (r * 0xffff) / a
			g = (g * 0xffff) / a
			b = (b * 0xffff) / a
			c := fmt.Sprintf("b=%02x%02x%02x,", r>>8, g>>8, b>>8)
			buf = append(buf, c...)
		}
	}

	if c.Format != "" {
		buf = append(buf, 'f', '=')
		buf = append(buf, c.Format...)
		buf = append(buf, ',')
	}
	if c.Quality != 0 {
		buf = append(buf, 'q', '=')
		buf = strconv.AppendInt(buf, int64(c.Quality), 10)
		buf = append(buf, ',')
	}
	if c.DisableOptimization {
		buf = append(buf, 'o', '=', '0', ',')
	}

	if len(buf) == 0 {
		return ""
	}
	return string(buf[:len(buf)-1])
}

func (a AspectMode) String() string {
	switch a {
	case AspectModeDefault:
		return "default"
	case AspectModeScale:
		return "scale"
	case AspectModeForceScale:
		return "force-scale"
	case AspectModePad:
		return "pad"
	}
	return ""
}

// URL returns the URL of the image.
func (img *Image) URL() *url.URL {
	p := img.Path
	if c := img.Config.String(); c != "" {
		p = path.Join("c", c, p)
	}

	return &url.URL{
		Scheme: "https",
		Host:   img.Proxy.Host,
		Path:   p,
	}
}

// SignedURL returns the URL of the image with the signature.
func (img *Image) SignedURL() *url.URL {
	u, s := img.urlAndSign()
	if s == "" {
		return u
	}

	if strings.HasPrefix(u.Path, "/c/") {
		u.Path = "/c/sig=" + s + "," + u.Path[len("/c/"):]
		return u
	}

	if strings.HasPrefix(u.Path, "/c!/") {
		u.Path = "/c!/sig=" + s + "," + u.Path[len("/c!/"):]
		return u
	}

	u.Path = "/c/sig=" + s + u.Path
	return u
}

// Sign returns the signature.
func (img *Image) Sign() string {
	_, s := img.urlAndSign()
	return s
}

func (img *Image) urlAndSign() (*url.URL, string) {
	u := img.URL()
	if img.Proxy == nil || img.Proxy.Secret == "" {
		return u, ""
	}

	p := u.Path
	if len(p) < 1 || p[0] != '/' {
		p = "/" + p
		u.Path = p
	}
	mac := hmac.New(sha256.New, []byte(img.Proxy.Secret))
	io.WriteString(mac, p)

	return u, "1." + base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func (img *Image) String() string {
	return img.URL().String()
}
