# goldmark-deferred-img64

A [goldmark](https://github.com/yuin/goldmark) extension to embed
base64-encoded images into rendered HTML, with content deferred into a
`<style>` block at the end of the document.

During markdown rendering, image references like:

```markdown
![a photo](path/to/image.jpg)
![a second photo](path/to/image2.jpg "optional image title")
```
are rendered as:
```html
<p>
  <img class="deferred_image_0"
    style="width:300px;height:500px"
    alt="a photo"
  >
  <img class="deferred_image_1"
    style="width:800px;height:100px"
    title="optional image title"
    alt="a second photo"
  >
</p>
<style data-deferredimg64>
  .deferred_image_0{ content:url(data:image/jpeg;base64,/9j/4A... /* bits constituting image.jpg */ ) }
  .deferred_image_1{ content:url(data:image/jpeg;base64,/9j/4AAQSkZ........ ) }
</style>
```

This produces fully self-contained single HTML file, while keeping the readable
content at the top of the document, and the large base64 blobs at the bottom.

This is inspired by
[`goldmark-img64`](https://github.com/tenkoh/goldmark-img64), which also embeds
local images as base64 data, though inlined directly into the `src` attribute
of each `<img>`. I *really like* the single-fetch-retrieves-everything aspect
of that approach, however:
 - Text and layout loading is delayed by large images early in the page.
 - Large data blobs make the HTML somewhat difficult to read and work with.
   - This is especially problematic when you're co-working with certain [blind
     idiot savants](https://code.claude.com/docs/en/overview), with a tendency
     to curl pages directly, lacking any post-processing step to render HTML
     into text.

`goldmark-deferred-img64` takes the same approach for reading and encoding
image data, but defers writing out of the main document flow, using a CSS
`content: url()` rule to set the `src` attribute of each `<img>` after loading.
`width` and `height` are set using an in-line `style` property to avoid any
layout shifts over the course of the load.

## Usage

```go
package main

import (
    "fmt"
    "bytes"
    "github.com/yuin/goldmark"
    "github.com/FraserLee/goldmark-deferred-img64"
)

func main() {

    source := []byte("![a photo](path/to/image.jpg)")

    md := goldmark.New(
        goldmark.WithExtensions(
          deferredimg64.New().
            WithScale(0.5).  // css width/height = 0.5 * image pixel dimensions
            WithBaseDir("/path/to/content/"), // defaults to pwd
        ),
    )

    var buf bytes.Buffer
    if err := md.Convert(source, &buf); err != nil {
        panic(err)
    }

    fmt.Println(buf.String())
}
```

## Caveats

CSS `content: url()` on `<img>` elements is [fairly well
supported](https://caniuse.com/mdn-css_properties_content_element_replacement)
at time of writing, working in:
- **Safari 9+** (2015)
- **Firefox 63+** (2018)
- **Chrome 28+** (2013)

However it's a bit of a weird feature, and it seems like there may have been
some ambiguity in the spec:
- [csswg-drafts#2831](https://github.com/w3c/csswg-drafts/issues/2831)
- [Firefox bug 215083](https://bugzilla.mozilla.org/show_bug.cgi?id=215083) (now closed, good historical reference)

Expect niche renderers to not love it.
