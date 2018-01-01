$(document).ready(function() {
    console.log("ready!");
    var canvas = document.getElementById("canvas");
    console.log(canvas)
    ctx = canvas.getContext('2d');

    mind.init({
        callback: function(robot) {
            skillID = "OpenCVSkill";
            robot.connectSkill({
                skillID: skillID,
                callback: robot.onRecvSkillData(function(skillID, data) {
                    console.log("received data from robot: ")
                    console.log(data)
                    if (data.length > 100) {
                        var img = $('<img id="dynamic" height="50px" width="50px" style="display:inline-block">'); //Equivalent: $(document.createElement('img'))
                        img.attr('src', 'data:image/jpeg;base64,' + data);
                        //var img = new Image();
                        img.src = 'data:image/jpeg;base64,' + data;
                        //ctx.drawImage(img, 0, 0);
                        img.appendTo($('#imagediv'));
                        document.getElementById('img').setAttribute('src', 'data:image/jpeg;base64,' + data);
                    } else {
                        //grab string and parse rect x,y,h,w
                        //draw on canvas
                    }
                })
            });
            document.getElementById("start").onclick = function() {
                robot.sendData({
                    skillID: skillID,
                    data: "start"
                })
            }
            document.getElementById("stop").onclick = function() {
                robot.sendData({
                    skillID: skillID,
                    data: "stop"
                })
            }
            document.getElementById("pic").onclick = function() {
                robot.sendData({
                    skillID: skillID,
                    data: "pic"
                })
            }
            document.getElementById("lookaround").onclick = function() {
                robot.sendData({
                    skillID: skillID,
                    data: "spinAround"
                })
            }
        }
    });
});
// setTimeout(() => {
//     robot.getInfo({
//         callback: function(info) { console.log("INFO: ", info) },
//         error: function(err) { console.error(err) }
//     })
// }, 10000)