# Fancy FS

A Go library for adding fancy additions to the filesystem.
This is useful for file managers and other tools that have a graphical representation of a users files and folders.

## Example

### Create an overlay image

This `over` image can be placed over a folder or other item indicating the type of content in a fancy folder.

```go
over := &canvas.Image{}

if ff, _ := fancyfs.DetailsForFolder(folderURI); ff != nil {
		if ff.BackgroundURI != nil {
			over.File = ff.BackgroundURI.Path()
		}
		if ff.BackgroundResource != nil {
			over.Resource = theme.NewColoredResource(ff.BackgroundResource, theme.ColorNameBackground)
		}
		s.over.FillMode = ff.BackgroundFill
	}
```
