package spracker

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

import (
	"github.com/moovweb/golog"
)

// We need to bundle an image with its name (derived from its filename).
type Image struct {
	Name string
	image.Image
}

// Bundle the name of a sprite image with its viewport into the spritesheet.
type Sprite struct {
	Name          string
	MagFactor     float64
	TopPadding    int
	BottomPadding int
	image.Rectangle
}

// Helper methods.
func (s *Sprite) Width() int {
	return s.Max.X - s.Min.X
}

func (s *Sprite) Height() int {
	return s.Max.Y - s.Min.Y
}

// Take a folder name, read all the images within, and return them in an array.
func ReadImageFolder(path string, log *golog.Logger) (images []Image, err error) {
	dir, err := os.Open(path)
	if err != nil {
		log.Errorf("Couldn't open folder '%s'", path)
		return nil, err
	} else {
		defer dir.Close()
	}
	dirInfo, err := dir.Stat()
	if err != nil {
		log.Errorf("Couldn't gather information for folder '%s'", path)
		return nil, err
	}
	if !dirInfo.IsDir() {
		// just ignore the request if it isn't a folder
		return nil, nil
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Errorf("Problem gathering names of image files in '%s'", path)
		return nil, err
	}

	// put these in a canonical order, otherwise it'll be platform-specific
	sort.Strings(names)

	images = make([]Image, 0)
	for _, name := range names {
		fullName := filepath.Join(path, name)
		imgFile, err := os.Open(fullName)
		if err != nil {
			log.Errorf("Couldn't open image file '%s'", fullName)
			continue
		} else {
			defer imgFile.Close()
		}
		imgInfo, err := imgFile.Stat()
		if err != nil {
			log.Errorf("Couldn't gather information for image file '%s'", fullName)
		}
		if imgInfo.IsDir() {
			// skip non-files
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".png" {
			img, err := png.Decode(imgFile)
			if err != nil {
				log.Errorf("Problem decoding png image in '%s'", fullName)
				continue
			}
			name = name[0 : len(name)-len(ext)]
			images = append(images, Image{name, img})
		} else if ext == ".jpg" || ext == ".jpeg" {
			img, err := jpeg.Decode(imgFile)
			if err != nil {
				log.Errorf("Problem decoding jpeg image in '%s'", fullName)
				continue
			}
			name = name[0 : len(name)-len(ext)]
			images = append(images, Image{name, img})
		} else {
			log.Debugf("Ignoring unrecognized file '%s'", fullName)
		}
	}

	if len(images) == 0 {
		log.Warningf("Folder '%s' contains no images; no sprite-sheet will be generated", path)
	}

	return images, nil
}

// Check whether a sprite name describes a hi-res (e.g. retina) sprite. If so,
// return the base name and magnification factor.
func IsMagnified(name string) (isIt bool, baseName string, factor float64) {
	segments := strings.Split(name, "@")
	if len(segments) != 2 {
		return false, name, 1
	}
	baseName = segments[0]
	factorStr := segments[1]
	if factorStr[len(factorStr)-1] == 'x' {
		f, err := strconv.ParseFloat(factorStr[0:len(factorStr)-1], 64)
		if err != nil {
			return false, name, 1
		}
		isIt = true
		factor = f
	} else {
		return false, name, 1
	}
	return
}

// Compose the spritesheet image and return an array of individual sprite data.
func GenerateSpriteSheet(images []Image, log *golog.Logger) (sheet draw.Image, sprites []Sprite) {
	var (
		sheetHeight int = 0
		sheetWidth  int = 0
	)
	sprites = make([]Sprite, 0)

	// calculate the size of the spritesheet and accumulate position and padding
	// data for the individual sprites within the sheet
	for _, img := range images {
		bounds := img.Bounds()

		_, name, factor := IsMagnified(img.Name)
		thisTopPadding := math.Ceil(factor)
		thisBottomPadding := thisTopPadding

		// see if the image dimensions are a multiple of the magnification factor
		fudge := 0
		if math.Mod(float64(bounds.Dy()), factor) != 0 {
			fudge = 1
			log.Warningf("Height of sprite `%s` (%dpx) is not a multiple of its magnification factor (%vx); rounding up to avoid truncated pixels.", name, bounds.Dy(), factor)
		}

		var prevBottomPadding float64
		if len(sprites) == 0 {
			prevBottomPadding = 0
			thisTopPadding = 0
		} else {
			prevBottomPadding = float64(sprites[len(sprites)-1].BottomPadding)
		}

		thisTopPadding = math.Max(thisTopPadding, prevBottomPadding)
		thisTopPaddingInt := int(thisTopPadding)
		thisBottomPaddingInt := int(thisBottomPadding)
		newSprite := Sprite{
			name,
			factor,
			thisTopPaddingInt,
			thisBottomPaddingInt,
			image.Rect(0, sheetHeight+thisTopPaddingInt-fudge, bounds.Dx(), sheetHeight+thisTopPaddingInt+bounds.Dy()),
		}
		sprites = append(sprites, newSprite)
		sheetHeight += bounds.Dy() + thisTopPaddingInt
		if bounds.Dx() > sheetWidth {
			sheetWidth = bounds.Dx()
		}
	}

	// create the sheet image
	sheet = image.NewRGBA(image.Rect(0, 0, sheetWidth, sheetHeight))

	// compose the sheet
	for i, img := range images {
		draw.Draw(sheet, sprites[i].Rectangle, img, image.Pt(0, 0), draw.Src)
	}

	return sheet, sprites
}

// Generate an array of SCSS variable definitions.
func GenerateScssVariables(folder string, sheetName string, sheetImg image.Image, sprites []Sprite) string {
	variables := make([]string, 0, len(sprites)+2)

	sheetUrl := fmt.Sprintf("$%s-url: url(\"%s.png\");\n", sheetName, folder+"/"+sheetName)
	variables = append(variables, sheetUrl)

	sheetWidth := fmt.Sprintf("$%s-width: %#vpx;", sheetName, sheetImg.Bounds().Max.X-sheetImg.Bounds().Min.X)
	sheetHeight := fmt.Sprintf("$%s-height: %#vpx;", sheetName, sheetImg.Bounds().Max.Y-sheetImg.Bounds().Min.Y)
	def := fmt.Sprintf("%s\n%s\n", sheetWidth, sheetHeight)
	variables = append(variables, def)

	for _, s := range sprites {
		name := s.Name
		factor := s.MagFactor
		prefix := fmt.Sprintf("%s-%s", sheetName, name)
		vX := fmt.Sprintf("$%s-x: %#vpx;", prefix, float64(-s.Min.X)/factor)
		vY := fmt.Sprintf("$%s-y: %#vpx;", prefix, float64(-s.Min.Y)/factor)
		vW := fmt.Sprintf("$%s-width: %#vpx;", prefix, float64(s.Width())/factor)
		vH := fmt.Sprintf("$%s-height: %#vpx;", prefix, float64(s.Height())/factor)
		def := fmt.Sprintf("%s\n%s\n%s\n%s\n", vX, vY, vW, vH)
		variables = append(variables, def)
	}

	return strings.Join(variables, "\n")
}

// Generate an array of SCSS mixin definitions.
const mixinFormat string = `@mixin %s-%s() {
  background: url("%s.png") no-repeat %#vpx %#vpx;%s
  width: %#vpx;
  height: %#vpx;
}
`

func GenerateScssMixins(folder string, sheetName string, sheetImg image.Image, sprites []Sprite) string {
	mixins := make([]string, 0, len(sprites))

	for _, s := range sprites {
		name := s.Name
		factor := s.MagFactor
		bgSize := ""
		if factor != 1 {
			bgSize = fmt.Sprintf("\n  @include background-size(%vpx %vpx);", float64(sheetImg.Bounds().Max.X)/factor, float64(sheetImg.Bounds().Max.Y)/factor)
		}
		def := fmt.Sprintf(mixinFormat, sheetName, name, folder+"/"+sheetName, float64(-s.Min.X)/factor, float64(-s.Min.Y)/factor, bgSize, float64(s.Width())/factor, float64(s.Height())/factor)
		mixins = append(mixins, def)
	}

	return strings.Join(mixins, "\n")
}

// Generate an array of CSS class definitions.
const classFormat string = `.%s-%s {
  background: url("%s.png") no-repeat %#vpx %#vpx;%s
  width: %#vpx;
  height: %#vpx;
}
`

func GenerateCssClasses(folder string, sheetName string, sheetImg image.Image, sprites []Sprite) string {
	classes := make([]string, 0, len(sprites))

	for _, s := range sprites {
		name := s.Name
		factor := s.MagFactor
		bgSize := ""
		if factor != 1 {
			bgSize = fmt.Sprintf("\n  background-size: %vpx %vpx;", float64(sheetImg.Bounds().Max.X)/factor, float64(sheetImg.Bounds().Max.Y)/factor)
		}
		class := fmt.Sprintf(classFormat, sheetName, name, folder+"/"+sheetName, float64(-s.Min.X)/factor, float64(-s.Min.Y)/factor, bgSize, float64(s.Width())/factor, float64(s.Height())/factor)
		classes = append(classes, class)
	}

	return strings.Join(classes, "\n")
}

func SpritesModified(folder, sheetFileName string) (bool, error) {
	sheetStat, err := os.Stat(sheetFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		} else {
			return false, err
		}
	}
	sheetModTime := sheetStat.ModTime()
	folderStat, err := os.Stat(folder)
	if err != nil {
		return false, err
	}
	folderModTime := folderStat.ModTime()
	if folderModTime.After(sheetModTime) {
		return true, nil
	}
	dir, err := os.Open(folder)
	if err != nil {
		return false, err
	} else {
		defer dir.Close()
	}
	imageNames, err := dir.Readdirnames(0)
	if err != nil {
		return false, err
	}
	for _, imgBaseName := range imageNames {
		imgPath := filepath.Join(folder, imgBaseName)
		imgStat, err := os.Stat(imgPath)
		if err != nil {
			continue
		}
		imgModTime := imgStat.ModTime()
		if imgModTime.After(sheetModTime) {
			return true, nil
		}
	}
	return false, nil
}

// Read a single folder containing sprites, and generate a single spritesheet
func GenerateSpriteSheetFromFolder(inputFolder, outputFolder, outputURI string, generateScss, checkTimeStamps bool, log *golog.Logger) (spriteSheet Image, styleSheet string, skipped bool, err error) {
	sheetName := filepath.Base(inputFolder)

	folder, ferr := os.Open(inputFolder)
	err = ferr
	if err != nil && os.IsNotExist(err) {
		log.Infof("Specified sprite folder '%s' does not exist; not generating anything", inputFolder)
		return
	} else if err != nil {
		log.Errorf("Couldn't open folder '%s' containing sprite images", inputFolder)
		return
	} else {
		defer folder.Close()
	}
	folderInfo, ferr := folder.Stat()
	err = ferr
	if err != nil {
		log.Errorf("Couldn't gather information for '%s'", inputFolder)
		return
	}
	if !folderInfo.IsDir() {
		log.Errorf("'%s' must be a folder", inputFolder)
		return
	}

	if checkTimeStamps {
		modified, ferr := SpritesModified(inputFolder, filepath.Join(outputFolder, sheetName+".png"))
		err = ferr
		if err != nil {
			log.Errorf("problem checking timestamps of '%s' and '%s.png'", inputFolder, sheetName)
			return
		}
		if !modified {
			log.Infof("no sprites have been added or modified since '%s.png' was generated; skipping generation", sheetName)
			skipped = true
			return
		}
	}

	images, ferr := ReadImageFolder(inputFolder, log)
	err = ferr
	if err != nil {
		log.Errorf("Problem reading sprite images in '%s'", inputFolder)
		return
	} else if len(images) > 0 {
		sheet, sprites := GenerateSpriteSheet(images, log)
		spriteSheet = Image{sheetName, sheet}
		vars := GenerateScssVariables(outputURI, sheetName, spriteSheet, sprites)
		mixins := GenerateScssMixins(outputURI, sheetName, spriteSheet, sprites)
		classes := GenerateCssClasses(outputURI, sheetName, spriteSheet, sprites)
		if generateScss {
			styleSheet = fmt.Sprintf("%s\n%s\n%s", vars, mixins, classes)
		} else {
			styleSheet = classes + "\n"
		}
		return
	} else {
		skipped = true
		return
	}
	return
}

// Read a folder containing subfolders which contain sprites, and generate a
// spritesheet for each subfolder.
func GenerateSpriteSheetsFromFolders(superFolder, outputFolder, outputURI string, generateScss, checkTimeStamps bool, log *golog.Logger) (spriteSheets []Image, styleSheets []string, err error) {

	container, err := os.Open(superFolder)
	if err != nil && os.IsNotExist(err) {
		log.Infof("Specified sprite folder '%s' does not exist; not generating anything", superFolder)
		return nil, nil, err
	} else if err != nil {
		log.Errorf("Couldn't open folder '%s' containing sprite subfolders", superFolder)
		return nil, nil, err
	} else {
		defer container.Close()
	}
	containerInfo, err := container.Stat()
	if err != nil {
		log.Errorf("Couldn't gather information for '%s'", superFolder)
		return nil, nil, err
	}
	if !containerInfo.IsDir() {
		log.Errorf("'%s' must be a folder", superFolder)
		return nil, nil, err
	}
	folders, err := container.Readdirnames(0)
	if err != nil {
		log.Errorf("Problem gathering names of sprite subfolders in '%s'", superFolder)
		return nil, nil, err
	}

	spriteSheets = make([]Image, 0)
	styleSheets = make([]string, 0)

	var anyErrors error = nil
	for _, folder := range folders {
		if checkTimeStamps {
			modified, err := SpritesModified(filepath.Join(superFolder, folder), filepath.Join(outputFolder, folder+".png"))
			if err != nil {
				log.Errorf("problem checking timestamps of '%s' and '%s.png'", folder, folder)
				anyErrors = err
				continue
			}
			if !modified {
				log.Infof("no sprites have been added or modified since '%s.png' was generated; skipping generation", folder)
				continue
			}
		}
		images, err := ReadImageFolder(filepath.Join(superFolder, folder), log)
		if err != nil {
			anyErrors = err
			continue
		} else if len(images) > 0 {
			sheet, sprites := GenerateSpriteSheet(images, log)
			vars := GenerateScssVariables(outputURI, folder, sheet, sprites)
			mixins := GenerateScssMixins(outputURI, folder, sheet, sprites)
			classes := GenerateCssClasses(outputURI, folder, sheet, sprites)
			spriteSheets = append(spriteSheets, Image{folder, sheet})
			if generateScss {
				styleSheets = append(styleSheets, fmt.Sprintf("%s\n%s\n%s", vars, mixins, classes))
			} else {
				styleSheets = append(styleSheets, classes+"\n")
			}
		}
		// else {
		// 	log.Warning("'%s' contains no images; no sprite-sheet generated", filepath.Join(superFolder, folder))
		// }
	}
	err = anyErrors

	return
}

// Write the spritesheet image to a file.
func WriteSpriteSheet(img image.Image, folder string, name string, log *golog.Logger) (err error) {
	dir, err := os.Open(folder)
	if err != nil {
		err = os.MkdirAll(folder, 0775)
		if err != nil {
			log.Errorf("Unable to create sprite-sheet output folder '%s'", folder)
			return err
		} else {
			log.Infof("Created sprite-sheet output folder '%s'", folder)
		}
	} else {
		defer dir.Close()
	}

	fullname := filepath.Join(folder, name+".png")
	os.Remove(fullname)
	outFile, err := os.Create(fullname)
	if err != nil {
		log.Errorf("Couldn't create sprite-sheet output file '%s'", fullname)
		return err
	} else {
		defer outFile.Close()
	}
	err = png.Encode(outFile, img)
	if err != nil {
		log.Warningf("Problem writing sprite-sheet to '%s'", fullname)
		return err
	}
	return nil
}

// Write a stylesheet string to a file.
func WriteStyleSheet(style string, folder string, baseName string, log *golog.Logger) (err error) {
	dir, err := os.Open(folder)
	if err != nil {
		err = os.MkdirAll(folder, 0775)
		if err != nil {
			log.Errorf("Unable to create stylesheet output folder '%s'", folder)
			return err
		} else {
			log.Infof("Created stylesheet output folder '%s'", folder)
		}
	} else {
		defer dir.Close()
	}

	fullname := filepath.Join(folder, baseName)
	os.Remove(fullname)
	outFile, err := os.Create(fullname)
	if err != nil {
		log.Errorf("Couldn't create SCSS output file '%s'", fullname)
		return err
	} else {
		defer outFile.Close()
	}
	_, err = outFile.WriteString(style)
	if err != nil {
		log.Errorf("Problem writing styles to '%s'", fullname)
		return err
	}
	return nil
}

// (grabbed off of Stack Overflow)
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
