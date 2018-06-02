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
	"strconv"
	"time"

	"github.com/lazywei/go-opencv/opencv"
)

func PitchTest() {
	log.Info.Println("PitchTest")
	hexabody.Stand()
	hexabody.Pitch(float64(GROUND_TO_FACE_PITCH_ANGLE), 100)
	// legs := hexabody.PitchRoll(angle, direction)
	// for i := 0; i < 6; i++ {
	// 	legs.SetLegPosition(i, legs[i])
	// }
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

func logger(msg string) {
	log.Info.Println("=================")
	log.Info.Println(msg)
	log.Info.Println("=================")
}

//===========================

//interval is number of 30 degree rotations from given view
func look(view View, interval int32) View {
	direction := LookAt2(view.direction+float64(SIZE_OF_INTERVAL_IN_DEGREES*interval), GROUND_TO_FACE_PITCH_ANGLE)
	image := TakePic()
	return NewView("Look-"+strconv.Itoa(int(direction)), image, direction, GROUND_TO_FACE_PITCH_ANGLE, time.Now())
}

func lookAtView(view View) View {
	hexabody.Stand()
	log.Info.Println("fn lookAtView")
	LookAt2(view.direction, view.angle)
	return view
}

func LookAt2(direction float64, angle float64) float64 {
	//log.Info.Println("look at called")
	//maybe run movements in parallel? closure is needed
	err := hexabody.MoveHead(direction, TIME_TO_COMPLETE_MOVEMENT)
	if err != nil {
		log.Error.Println("Move head failed")
		return -1
	}
	hexabody.Pitch(float64(GROUND_TO_FACE_PITCH_ANGLE), TIME_TO_COMPLETE_MOVEMENT)
	time.Sleep(TIME_TO_SLEEP_AFTER_MOVEMENT_IN_MS) //first picture blurry, others seem good.
	return direction
}

func ContainsFace(image *image.RGBA) bool {
	var cvimg *opencv.IplImage
	var faces []*opencv.Rect
	if image == nil {
		log.Error.Println("NO IMAGE")
	}
	cvimg = opencv.FromImage(image)
	if cvimg == nil {
		log.Error.Println("NO CVIMG")
	}
	faces = opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml").DetectObjects(cvimg)
	if len(faces) > 0 {
		return true
	}
	return false

}
