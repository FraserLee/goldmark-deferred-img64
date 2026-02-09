// goldmark extension that embeds images as base64 data URIs via CSS
// `content: url()`s, in a `<style>` block at the end of HTML.
package deferredimg64

import (
    "bytes"
    "encoding/base64"
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "os"
    "path/filepath"
    "strings"

    "github.com/gabriel-vasile/mimetype"
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/renderer"
    "github.com/yuin/goldmark/util"
)

type deferredImage struct {
    class   string
    dataURI string
}

type Extension struct {
    scaleFactor float64
    baseDir     string
    images      []deferredImage
}

type imgRenderer struct {
    e *Extension
}

func New() *Extension {
    return &Extension{scaleFactor: 1}
}

// All images are given inline CSS styles, setting
// `... set style="width:{width}px; height:{height}px">`
// where width and height are the dimensions of the image, multiplied by scaleFactor.
// (scaleFactor = 0.5 halves the pixel dimensions, appropriate for retina displays)
func (e *Extension) WithScale(factor float64) *Extension {
    e.scaleFactor = factor
    return e
}

func (e *Extension) WithBaseDir(dir string) *Extension {
    e.baseDir = dir
    return e
}

func (e *Extension) Extend(m goldmark.Markdown) {
    m.Renderer().AddOptions(renderer.WithNodeRenderers(
        util.Prioritized(&imgRenderer{e}, 500), // default has priority 1000
    ))
}

func (r *imgRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
    reg.Register(ast.KindImage, r.renderImage)
    reg.Register(ast.KindDocument, r.renderDocument)
}


// delegates alt text rendering to goldmark's default text/string renderers via
// WalkContinue.
// Caveat: inline formatting in alt text (e.g., ![**bold**](img.png)) will
// cause HTML tags to be emitted inside the alt attribute. If this is an issue,
// could either fix it by exporting renderTexts (html.go:773), or by
// reimplementing it.
func (r *imgRenderer) renderImage(
    w util.BufWriter,
    source []byte,
    node ast.Node,
    entering bool,
) (ast.WalkStatus, error) {
    if entering {
        n := node.(*ast.Image)
        _, _ = w.WriteString(`<img`)
        if class, width, height, ok := r.embedImage(n); ok {
            fmt.Fprintf(w, ` class="%s" style="width:%gpx;height:%gpx"`, class, width, height)
        } else {
            _, _ = w.WriteString(` src="`)
            _, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
            _ = w.WriteByte('"')
        }

        if n.Title != nil {
            _, _ = w.WriteString(` title="`)
            _, _ = w.Write(util.EscapeHTML(n.Title))
            _ = w.WriteByte('"')
        }

        _, _ = w.WriteString(` alt="`)

    } else {
        _, _ = w.WriteString(`">`)
    }
    return ast.WalkContinue, nil
}

// embedImage tries to read, encode, and store the image data. Returns the
// CSS class name & scaled dimensions on success.
func (r *imgRenderer) embedImage(n *ast.Image) (
    class string,
    width, height float64,
    ok bool,
) {
    src := string(n.Destination)
    if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
        // TODO: look into fetching remote images (should be configurable)
        return
    }

    path := filepath.Clean(src)
    if !filepath.IsAbs(path) && r.e.baseDir != "" {
        path = filepath.Join(r.e.baseDir, path)
    }
    b, err := os.ReadFile(path)
    if err != nil {
        return
    }

    mtype := mimetype.Detect(b).String()

    cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
    if err != nil {
        return
    }

    var dataBuf bytes.Buffer
    fmt.Fprintf(&dataBuf, "data:%s;base64,", mtype)
    enc := base64.NewEncoder(base64.StdEncoding, &dataBuf)
    _, _ = enc.Write(b)
    enc.Close()

    class = fmt.Sprintf("deferred_image_%d", len(r.e.images))
    r.e.images = append(r.e.images, deferredImage{
        class:   class,
        dataURI: dataBuf.String(),
    })

    width = float64(cfg.Width) * r.e.scaleFactor
    height = float64(cfg.Height) * r.e.scaleFactor
    ok = true
    return
}

// On exit, print a <style> block embedding image content
func (r *imgRenderer) renderDocument(
    w util.BufWriter,
    _ []byte,
    _ ast.Node,
    entering bool,
) (ast.WalkStatus, error) {
    if entering {
        r.e.images = nil
    } else if len(r.e.images) > 0 {
        _, _ = w.WriteString("<style data-deferredimg64>\n")
        for _, img := range r.e.images {
            fmt.Fprintf(w, ".%s{content:url(%s)}\n", img.class, img.dataURI)
        }
        _, _ = w.WriteString("</style>")
    }
    return ast.WalkContinue, nil
}

