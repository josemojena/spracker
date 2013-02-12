package main

import (
  "fmt"
  "math"
  "math/rand"
  "time"
)

func main() {

  // var (
  //   generateScss    bool
  //   checkTimestamps bool
  //   spritesFolder      string
  //   spriteSheetsFolder string
  //   styleSheetsFolder  string
  // )

  // flag.BoolVar(&generateScss, "scss", true, "generate Sass/SCSS variables and mixins")
  // flag.BoolVar(&checkTimestamps, "check-timestamps", true, "don't regenerate sprite-sheets if they're newer than their component sprite images")
  // flag.StringVar(&spritesFolder, "sprites-folder", ".", "input folder containing subfolders with sprite images")
  // flag.StringVar(&spriteSheetsFolder, "spritesheets-folder", ".", "output folder in which to deposit the sprite-sheets")
  // flag.StringVar(&styleSheetsFolder, "stylesheets-folder", ".", "output folder in which to deposit the stylesheets")

  // flag.Parse()

  // log := golog.NewLogger("")
  // log.AddProcessor("first", golog.NewConsoleProcessor(golog.LOG_INFO, true))

  // var stylesheetExtension string
  // if generateScss {
  //   stylesheetExtension = ".scss"
  // } else {
  //   stylesheetExtension = ".css"
  // }

  // sheets, styles, _ := spracker.GenerateSpriteSheetsFromFolders(spritesFolder, spriteSheetsFolder, spriteSheetsFolder, generateScss, checkTimestamps, log)
  // for i, sheet := range sheets {
  //   wstErr := spracker.WriteSpriteSheet(sheet.Image, spriteSheetsFolder, sheet.Name, log)
  //   if wstErr == nil {
  //     log.Info("Generated sprite-sheet '%s.png'", filepath.Join(spriteSheetsFolder, sheet.Name))
  //   }
  //   wspErr := spracker.WriteStyleSheet(styles[i], styleSheetsFolder, sheet.Name+stylesheetExtension, log)
  //   if wspErr == nil {
  //     log.Info("Generated stylesheet '%s%s'", filepath.Join(styleSheetsFolder, sheet.Name), stylesheetExtension)
  //   }
  // }

  // golog.FlushLogsAndDie()


  // for i := 0; i < 10; i++ {
  //   size    := 64 + rand.Int() % 20
  //   magFrac := float64(rand.Int() % 10) / 10
  //   magFac  := rand.Int() % 4 + magFrac
  //   width  := baseSize + variance
  //   height := baseSize + variance
  //   imgResp := http.Get(fmt.Sprintf("http://placekitten.com/%d/%d", width, height))
  // }


  rand.Seed(time.Now().UnixNano())

  size    := 64 + rand.Int() % 20
  magFrac := float64(rand.Int() % 2 * 5) / 10
  magFac  := float64(rand.Int() % 4 + 1) + magFrac

  size = int(math.Floor(magFac)) * (64 + rand.Int() % 20)

  fmt.Println(size, magFrac, magFac)
}
