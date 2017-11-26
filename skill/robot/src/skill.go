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
	"mind/core/framework/skill"
	"os"

	"github.com/lazywei/go-opencv/opencv"
)

type FollowSkill struct {
	skill.Base
	state   FollowState
	stop    chan bool
	cascade *opencv.HaarCascade
}

func NewSkill() skill.Interface {
	return &FollowSkill{
		state:   FollowState{"idle"},
		stop:    make(chan bool),
		cascade: opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml"),
	}
}

type FollowState struct {
	state string
}

type View struct {
	image     *image.RGBA
	direction float64
	angle     float64
}

/*
EVENTS
*/

func (d *FollowSkill) OnStart() {
	log.Info.Println("Started")
	hexabody.Start()
	if !media.Available() {
		log.Error.Println("Media driver not available")
		return
	}
	if err := media.Start(); err != nil {
		log.Error.Println("Media driver could not start")
	}
}

func (d *FollowSkill) OnClose() {
	hexabody.Close()
}

func (d *FollowSkill) OnDisconnect() {
	os.Exit(0) // Closes the process when remote disconnects
}

func (d *FollowSkill) OnRecvString(data string) {
	log.Info.Println(data)
	switch data {
	case "start":
		go d.sight()
		break
	case "stop":
		d.stop <- true
		break
	case "pic":
		go TakePic()
		break
	case "spinAround":
		go d.Follow()
		break
	}
}

/*
End Events
*/

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

func SpinAround(c chan View) {
	log.Info.Println("fn spinAround")
	intervals := 12
	degreesPerInterval := float64(360 / intervals)
	pitch := 10.0
	TTC := 50
	looking := true
	hexabody.Spin(0, TTC)
	log.Info.Println(hexabody.Direction())
	for looking {
		hexabody.Spin(degreesPerInterval, TTC)
		log.Info.Println("turning head ")
		direction := hexabody.Direction()
		log.Info.Println(direction)
		LookAt2(direction, pitch)
		image := TakePic()
		c <- View{image, direction, pitch}
		if direction >= 360 {
			looking = false
		}
	}
}

func LookAround(c chan View) {
	log.Info.Println("fn spinAround")
	intervals := 12
	degreesPerInterval := float64(360 / intervals)
	pitch := 10.0
	TTC := 50
	looking := true
	hexabody.MoveHead(0, TTC)
	log.Info.Println(hexabody.Direction())
	for looking {
		hexabody.MoveHead(degreesPerInterval, TTC)
		log.Info.Println("turning head ")
		direction := hexabody.Direction()
		log.Info.Println(direction)
		LookAt2(direction, pitch)
		image := TakePic()
		c <- View{image, direction, pitch}
		if direction >= 360 {
			looking = false
		}
	}
}

func Idle() {
	hexabody.StopPitch()
	hexabody.MoveHead(0, 300)
}

func Reset() {
	hexabody.StopPitch()
	hexabody.MoveHead(0, 300)
}

func LookAt(direction float64, angle float64) {
	log.Info.Println("look at called")
	log.Info.Println(direction)
	hexabody.Spin(0, 300)
	hexabody.Spin(direction, 300)
	hexabody.Pitch(angle, 300)
}

func LookAt2(direction float64, angle float64) {
	log.Info.Println("look at called")
	hexabody.MoveHead(direction, 10)
	legs := hexabody.PitchRoll(direction, angle)
	for i := 0; i < 6; i++ {
		legs.SetLegPosition(i, legs[i])
	}
}

func (d *FollowSkill) ContainsFace(view View) bool {
	log.Info.Println("checking for faces")
	cvimg := opencv.FromImageUnsafe(view.image)
	faces := d.cascade.DetectObjects(cvimg)

	if len(faces) > 0 {
		log.Info.Println("face(s) found")
		log.Info.Println(len(faces))
		return true
	} else {
		log.Info.Println("no faces found")
		return false
	}

}

func (d *FollowSkill) FindFaces(c1 chan View, c2 chan View) {
	log.Info.Println("fn findFaces")
	for {
		cV := <-c1
		if d.ContainsFace(cV) {
			log.Info.Println("face found")
			SendImage(cV.image)
			c2 <- cV
		}

	}
}

func (d *FollowSkill) sight() {
	for {
		select {
		case <-d.stop:
			return
		default:
			image := media.SnapshotRGBA()
			buf := new(bytes.Buffer)
			jpeg.Encode(buf, image, nil)
			str := base64.StdEncoding.EncodeToString(buf.Bytes())
			framework.SendString(str)
			cvimg := opencv.FromImageUnsafe(image)
			faces := d.cascade.DetectObjects(cvimg)
			//facesStringv := base64.StdEncoding.EncodeToString(faces)
			//framework.send(facesString)
			hexabody.StandWithHeight(float64(len(faces)) * 50)
			log.Info.Println(faces)
		}
	}
}

func (d *FollowSkill) Follow() {
	log.Info.Println("fn follow called")
	channel1 := make(chan View, 10)
	channel2 := make(chan View, 10)
	StartingDirection := hexabody.Direction()
	log.Info.Println(StartingDirection)
	go LookAround(channel1)
	go d.FindFaces(channel1, channel2)

	for {
		cV := <-channel2
		log.Info.Println("LookAt")
		LookAt(StartingDirection, 0)
		LookAt(cV.direction, cV.angle)
		hexabody.Walk(cV.direction, 5000)
		//close channel if found? stop go routines to see if look at works
	}
	//search
	//moveTowards
	//maintainDistance
}
