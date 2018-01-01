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
	"sync"

	"github.com/lazywei/go-opencv/opencv"
)

type View struct {
	id        int
	image     *image.RGBA
	direction float64
	angle     float64
}

type FollowSkill struct {
	skill.Base
	state          FollowState
	stop           chan bool
	allViews       chan View
	viewsWithFaces chan View
	wg             sync.WaitGroup
	cascade        *opencv.HaarCascade
}

func NewSkill() skill.Interface {
	return &FollowSkill{
		state:          FollowState{"idle"},
		stop:           make(chan bool),
		allViews:       make(chan View, 1000),
		viewsWithFaces: make(chan View),
		cascade:        opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml"),
	}
}

type FollowState struct {
	state string
}

/*
EVENTS
*/

func (FS *FollowSkill) OnStart() {
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

func (FS *FollowSkill) OnClose() {
	hexabody.Close()
}

func (FS *FollowSkill) OnDisconnect() {
	os.Exit(0) // Closes the process when remote disconnects
}

func (FS *FollowSkill) OnRecvString(data string) {
	log.Info.Println(data)
	switch data {
	case "start":
		go FS.sight()
		break
	case "stop":
		FS.stop <- true
		break
	case "pic":
		go TakePic()
		break
	case "spinAround":
		go FS.Follow()
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
		c <- View{1, image, direction, pitch}
		if direction >= 360 {
			looking = false
		}
	}
}

/*
LookAround
State: working
Description: turn head in 360 and take pictures which are pushed into the all views channel
*/
func (FS *FollowSkill) LookAround() {
	//define
	intervals := 12
	currentInterval := 0
	degreesPerInterval := float64(360 / intervals)
	pitch := 10.0
	//init
	hexabody.Stand()
	direction := LookAt2(float64(currentInterval), pitch)
	if direction == -1 {
		log.Error.Println("look at failed")
		return
	}
	for {
		select {
		case <-FS.stop:
			log.Info.Println("stop received")
			break
		default:
			direction = LookAt2(degreesPerInterval*float64(currentInterval), pitch)
			if direction == -1 {
				log.Error.Println("look at failed")
				break
			}
			image := TakePicAndSend()
			FS.allViews <- View{currentInterval, image, direction, pitch}
			log.Info.Println("Current Interval: ", currentInterval)
			currentInterval = currentInterval + 1
			if currentInterval == intervals {
				currentInterval = 0
				direction = LookAt2(float64(currentInterval), pitch)
				if direction == -1 {
					log.Error.Println("look at failed")
				}
				FS.wg.Done()
				return
			}
		}
	}
	logger("LookAround Complete")
	return
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

func lookAtView(view View) {
	log.Info.Println("fn lookAtView")
	LookAt2(view.direction, view.angle)
}

func LookAt2(direction float64, angle float64) float64 {
	log.Info.Println("look at called")
	err := hexabody.MoveHead(direction, 50)
	if err != nil {
		log.Error.Println("Move head failed")
		return -1
	}
	legs := hexabody.PitchRoll(direction, angle)
	for i := 0; i < 6; i++ {
		legs.SetLegPosition(i, legs[i])
	}
	return direction
}

func (FS *FollowSkill) ContainsFace(view View) bool {
	log.Info.Println("checking for faces")
	cvimg := opencv.FromImageUnsafe(view.image)
	faces := FS.cascade.DetectObjects(cvimg)
	log.Info.Println("face detection complete")
	if len(faces) > 0 {
		log.Info.Println("face(s) found. ", "View: ", view.id)
		log.Info.Println(len(faces))
		return true
	}
	log.Info.Println("no faces found. ", "View: ", view.id)
	return false

}

func (FS *FollowSkill) FindFaces() {
	log.Info.Println("fn findFaces")
	for {
		select {
		case <-FS.stop:
			break
		case currentView := <-FS.allViews:
			if FS.ContainsFace(currentView) {
				log.Info.Println("!!!!!!!Face found!!!!!!")
				log.Info.Println("View: ", currentView.id)
				FS.viewsWithFaces <- currentView
				return
				//SendImage(currentView.image)
			}
		}

	}
	logger("FindFaces Complete")
	return
}

//Follow Skill is the entry point for this file
func (FS *FollowSkill) Follow() {
	if FS == nil {
		log.Info.Println("no follow skill")
	}
	log.Info.Println("fn follow called")
	StartingDirection := hexabody.Direction()
	log.Info.Println("Starting Direction: ", StartingDirection)
	FS.wg.Add(1)
	go FS.LookAround()
	FS.wg.Wait()
	log.Info.Println("starting find faces")
	go FS.FindFaces()
	//FS.wg.Wait()

	for {
		select {
		case <-FS.stop:
			logger("stop called")
			break //
		case viewWithFace := <-FS.viewsWithFaces:
			FS.stop <- true
			close(FS.allViews)
			close(FS.viewsWithFaces)
			log.Info.Println("looking at view: ", viewWithFace.id)
			lookAtView(viewWithFace)
			//hexabody.Walk(viewWithFace.direction, 5000)
			//close channel if found? stop go routines to see if look at works
		}
	}
	//search
	//moveTowards
	//maintainDistance
}

func logger(msg string) {
	log.Info.Println("=================")
	log.Info.Println(msg)
	log.Info.Println("=================")
}

func (FS *FollowSkill) sight() {
	for {
		select {
		case <-FS.stop:
			return
		default:
			image := media.SnapshotRGBA()
			buf := new(bytes.Buffer)
			jpeg.Encode(buf, image, nil)
			str := base64.StdEncoding.EncodeToString(buf.Bytes())
			framework.SendString(str)
			cvimg := opencv.FromImageUnsafe(image)
			faces := FS.cascade.DetectObjects(cvimg)
			//facesStringv := base64.StdEncoding.EncodeToString(faces)
			//framework.send(facesString)
			hexabody.StandWithHeight(float64(len(faces)) * 50)
			log.Info.Println(faces)
		}
	}
}
