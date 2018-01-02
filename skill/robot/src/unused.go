package examples

// func SpinAround(c chan View) {
// 	log.Info.Println("fn spinAround")
// 	intervals := 12
// 	degreesPerInterval := float64(360 / intervals)
// 	TTC := 50
// 	looking := true
// 	hexabody.Spin(0, TTC)
// 	log.Info.Println(hexabody.Direction())
// 	for looking {
// 		hexabody.Spin(degreesPerInterval, TTC)
// 		log.Info.Println("turning head ")
// 		direction := hexabody.Direction()
// 		log.Info.Println(direction)
// 		LookAt2(direction, GROUND_TO_FACE_PITCH_ANGLE)
// 		image := TakePic()
// 		c <- View{"SpinAround-" + strconv.FormatFloat(direction, 'E', -1, 64), image, direction, GROUND_TO_FACE_PITCH_ANGLE, time.Now()}
// 		if direction >= 360 {
// 			looking = false
// 		}
// 	}
// }

// func (FS *FollowSkill) FollowSync() {
// 	if FS == nil {
// 		log.Info.Println("no follow skill")
// 	}
// 	//log.Info.Println("fn follow called")
// 	StartingDirection := hexabody.Direction()
// 	log.Info.Println("Starting Direction: ", StartingDirection)
// 	//FS.wg.Add(1)
// 	FS.LookAround()
// 	//FS.wg.Wait()
// 	log.Info.Println("starting find faces")
// 	//FS.wg.Add(1)
// 	FS.FindFaces()
// 	//FS.wg.Wait()
// 	log.Info.Println("here")

// 	log.Info.Println("starting look look at found face")
// 	for {
// 		select {
// 		case <-FS.stop:
// 			logger("stop called")
// 			return
// 			break
// 		case viewWithFace := <-FS.viewsWithFaces:
// 			//FS.stop <- true
// 			//close(FS.allViews)
// 			//close(FS.viewsWithFaces)
// 			log.Info.Println("looking at view: ", viewWithFace.id)
// 			lookAtView(viewWithFace)
// 			//hexabody.Walk(viewWithFace.direction, 5000)
// 			//close channel if found? stop go routines to see if look at works
// 		}
// 	}
// 	//search
// 	//moveTowards
// 	//maintainDistance
// }
