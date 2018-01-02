package examples

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"mind/core/framework"
	"mind/core/framework/drivers/hexabody"
	"mind/core/framework/drivers/media"
	"mind/core/framework/log"
)

func helptest() {
	log.Info.Println("helper working")
}

func SendImage(image *image.RGBA) {
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, image, nil)
	str := base64.StdEncoding.EncodeToString(buf.Bytes())
	framework.SendString(str)
}

func TakePic() *image.RGBA {
	log.Info.Println("taking photo")
	image := media.SnapshotRGBA()
	return image
}

func TakePicAndSend() *image.RGBA {
	pic := TakePic()
	SendImage(pic)
	return pic
}

func Idle() {
	hexabody.StopPitch()
	hexabody.MoveHead(0, 300)
}

func Reset() {
	hexabody.StopPitch()
	hexabody.MoveHead(0, 300)
}
