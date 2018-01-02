package examples

import (
	"bytes"
	"encoding/base64"
	"image/jpeg"
	"math"
	"mind/core/framework"
	"mind/core/framework/drivers/hexabody"
	"mind/core/framework/drivers/media"
	"mind/core/framework/log"
	"mind/core/framework/skill"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lazywei/go-opencv/opencv"
)

const ALL_VIEWS_BUFFER_SIZE = 1000
const VIEW_EXPIRATION_IN_SECONDS = 300
const SIZE_OF_INTERVAL_IN_DEGREES = 60.0
const INTERVALS = 360 / SIZE_OF_INTERVAL_IN_DEGREES
const TIME_TO_COMPLETE_MOVEMENT = 200
const TIME_TO_SLEEP_AFTER_MOVEMENT_IN_MS = time.Millisecond * 200
const GROUND_TO_FACE_PITCH_ANGLE = 20.0

type FollowSkill struct {
	skill.Base
	state          FollowState
	stop           chan bool
	allViews       chan View
	viewsWithFaces chan View
	adjustView     chan View
	wg             sync.WaitGroup
	cascade        *opencv.HaarCascade
}

func NewSkill() skill.Interface {
	return &FollowSkill{
		state:          FollowState{"idle"},
		stop:           make(chan bool),
		allViews:       make(chan View, ALL_VIEWS_BUFFER_SIZE),
		viewsWithFaces: make(chan View),
		adjustView:     make(chan View),
		cascade:        opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml"),
	}
}

type FollowState struct {
	currState string
}

/*====================================================
EVENTS
=====================================================*/

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
		PitchTest()
		break
	case "spinAround":
		go FS.FollowAsync()
		break
	}
}

/*====================================================
End Events
====================================================*/

/*
LookAround
State: working
Description: turn head in 360 and take pictures which are pushed into the all views channel
*/
func (FS *FollowSkill) LookAround() {
	currentInterval := 0

	direction := LookAt2(0.0, GROUND_TO_FACE_PITCH_ANGLE) // start from same spot everytime
	if direction == -1 {
		log.Error.Println("look at failed")
		return
	}
	for FS.state.currState == "searching" {
		select {
		case <-FS.stop:
			log.Info.Println("stop received")
			break
		default:
			FS.allViews <- NewView("LookAround-"+strconv.Itoa(int(direction)), direction, GROUND_TO_FACE_PITCH_ANGLE)
			// FS.allViews <- View{"LookAround-" + strconv.Itoa(int(direction)), image, direction, GROUND_TO_FACE_PITCH_ANGLE, time.Now()}

			if math.Mod(float64(currentInterval), float64(INTERVALS)) == (INTERVALS - 1) {
				return
			}
			currentInterval = currentInterval + 1
			direction = LookAt2(SIZE_OF_INTERVAL_IN_DEGREES*float64(currentInterval), GROUND_TO_FACE_PITCH_ANGLE)
			if direction == -1 {
				log.Error.Println("look at failed")
				break
			}

		}
	}
	logger("LookAround Complete")
	return
}

//interval is number of 30 degree rotations from given view
func look(view View, interval int32) View {
	direction := LookAt2(view.direction+float64(SIZE_OF_INTERVAL_IN_DEGREES*interval), GROUND_TO_FACE_PITCH_ANGLE)
	return NewView("Look-"+strconv.Itoa(int(direction)), direction, GROUND_TO_FACE_PITCH_ANGLE)

	//FS.allViews <- View{0,image,newDirection,10}
}

func LookAt(direction float64, angle float64) {
	log.Info.Println("look at called")
	log.Info.Println(direction)
	hexabody.Spin(0, 300)
	hexabody.Spin(direction, 300)
	hexabody.Pitch(angle, 300)
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

func (FS *FollowSkill) ContainsFace(view View) bool {
	log.Info.Println("Time since captured: ", time.Now().Sub(view.timestamp))
	var cvimg *opencv.IplImage
	var faces []*opencv.Rect
	if view.image == nil {
		log.Error.Println("NO IMAGE")
	}
	cvimg = opencv.FromImage(view.image)
	if cvimg == nil {
		log.Error.Println("NO CVIMG")
	}
	faces = FS.cascade.DetectObjects(cvimg)
	if len(faces) > 0 {
		return true
	}
	return false

}

func ContainsFaceAsync(view View, vFW chan<- View) {
	log.Info.Println("Time since captured: ", time.Now().Sub(view.timestamp))
	var cvimg *opencv.IplImage
	var faces []*opencv.Rect
	if view.image == nil {
		log.Error.Println("NO IMAGE")
	}
	cvimg = opencv.FromImage(view.image)
	if cvimg == nil {
		log.Error.Println("NO CVIMG")
	}

	faces = opencv.LoadHaarClassifierCascade("assets/haarcascade_frontalface_alt.xml").DetectObjects(cvimg) //maybe cant be in go routine?
	if len(faces) > 0 {
		log.Info.Println("******Face found at ", "view: ", view.name+"-", view.direction)
		SendImage(view.image)
		vFW <- view
		//FS.state.currState = "following"
	}
	log.Info.Println("no faces found in ", view.id)
}

func (FS *FollowSkill) FindFaces() {
	for {
		select {
		case <-FS.stop:
			log.Info.Println("stop called during find faces")
			return
		case currentView := <-FS.allViews:
			if int(time.Now().Sub(currentView.timestamp).Seconds()) > VIEW_EXPIRATION_IN_SECONDS {
				log.Info.Println("5 minutes since taken, image has expired")
				break
			}
			cV := currentView
			go func() {
				ContainsFaceAsync(cV, FS.viewsWithFaces)
			}()
			// if FS.ContainsFace(currentView) {
			// 	log.Info.Println("******Face found at ", "view: ", currentView.id+"-", currentView.direction)
			// 	SendImage(currentView.image)
			// 	FS.viewsWithFaces <- currentView
			// 	log.Info.Println("returning from find faces")
			// 	FS.state.currState = "following"
			// 	return
			// }
		}
	}
}

func (FS *FollowSkill) CheckPeripherals(view View) {
	log.Info.Println("Adjusting direction")
	if FS.ContainsFace(look(view, 1)) == true {
		log.Info.Println("success on look left")
		FS.viewsWithFaces <- look(view, 1)
	} else if FS.ContainsFace(look(view, -1)) == true {
		log.Info.Println("success on look right")
		FS.viewsWithFaces <- look(view, -1)
	} else {
		log.Info.Println("could not relocate face")
		go FS.FindFaces()
	}
}

func (FS *FollowSkill) ConfirmFaceFound() {
	for {
		select {
		case <-FS.stop:
			logger("stop called")
			return
		case viewWithFace := <-FS.viewsWithFaces:
			log.Info.Println("looking at view: ", viewWithFace.id)
			look(viewWithFace, 0)
			log.Info.Println("calculated direction: ", viewWithFace.direction, " API direction: ", hexabody.Direction())
			image := TakePicAndSend()
			lastView := View{viewWithFace.id, "ConfirmFaceFound-" + strconv.Itoa(int(viewWithFace.direction)), image, viewWithFace.direction, viewWithFace.angle, time.Now()}
			if FS.ContainsFace(lastView) {
				log.Info.Println("Success!")
				hexabody.StopPitch()
				hexabody.Walk(lastView.direction, 3000)
			} else {
				FS.CheckPeripherals(lastView)
			}
			//os.Exit(0)
			//hexabody.Walk(viewWithFace.direction, 5000)
			//close channel if found? stop go routines to see if look at works
		}
	}
}

//Follow Skill is the entry point for this file
func (FS *FollowSkill) FollowAsync() {
	if FS == nil {
		log.Info.Println("no follow skill")
	}
	StartingDirection := hexabody.Direction()
	log.Info.Println("Current Direction: ", StartingDirection)
	FS.state.currState = "searching"
	go FS.LookAround()
	go FS.FindFaces()
	go FS.ConfirmFaceFound()
	//search
	//moveTowards
	//maintainDistance
}

func PitchTest() {
	log.Info.Println("PitchTest")
	hexabody.Stand()
	hexabody.Pitch(float64(GROUND_TO_FACE_PITCH_ANGLE), 100)
	// legs := hexabody.PitchRoll(angle, direction)
	// for i := 0; i < 6; i++ {
	// 	legs.SetLegPosition(i, legs[i])
	// }
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
