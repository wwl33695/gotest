package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"rtcp"
	//	"rtp"
	"sip"
	"strconv"
	"strings"
	"time"
)

const (
	UNKNOWN = iota
	LOGIN
	REGISTER
	QUERYDEVICE
	LIVEPLAY
	BREAKPLAY
	LOGOUT
)

//WWW-Authenticate: Digest nonce="13133944849:f251436a279f25d0b879d2813df6b5b2",algorithm=MD5,realm="1100000000"

//15010000004000000001:1100000000:123456  9E15EFC22491F20D9244C5B6332A334E
//13133944849:f251436a279f25d0b879d2813df6b5b2
//REGISTER:sip:1100000000 E29314BF063AA97B770436FAC1AE466A
var databuffer [2000 * 1024]byte
var offset int = 0
var chandata chan []byte

func parsestring() {
	ttt := "gb28181://192.168.1.176:5065:15010000004000000001:123456@192.168.6.105:5060:11000000002000000001/34020000001310000001";

	pos := strings.Index(ttt, "//")
	temp := ttt[pos+2:]
	pos = strings.Index(temp, ":")

	println(temp[:pos])

	pos++
	pos1 := strings.Index(temp[pos:], ":")
//	println(pos, pos1)
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	pos1 = strings.Index(temp[pos:], ":")
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	pos1 = strings.Index(temp[pos:], "@")
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	pos1 = strings.Index(temp[pos:], ":")
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	pos1 = strings.Index(temp[pos:], ":")
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	pos1 = strings.Index(temp[pos:], "/")
//	println(pos, pos1)
	println(temp[pos:pos+pos1])

	pos = pos + pos1 + 1
	println(temp[pos:])
}

func test() {

//	buf := make([]byte, 1024)

	for {
		select {
			case buf := <- ttt:
				println("recv data from channel")
				for i:= 0;i<len(buf);i++ {
					println(buf[i])
				}
			default:
				println("test default")
				time.Sleep(time.Second * 1)
		}
	}
}

var ttt chan []byte
func main() {
	var nums []byte = []byte{1,2,3,4,5,6,7,8,9,10}

	ttt = make(chan []byte, 1024)
	go test()

	i := 0;
	for {
		i += 3
		if i >= len(nums) {
			i = 1
		}

		select {
			case ttt <- nums[0:i]:
//				println("send data to channel")
			default:
				println("main default")
				time.Sleep(time.Second * 1)
		}		
//		time.Sleep(time.Second * 1)
//		println("aaaaaa")
	}
//	parsestring()
//	return

	uac := &sip.UASInfo{
		ServerID:   "11000000002000000001",
		ServerIP:   "192.168.6.105",
		ServerPort: "5060",
		UserName:   "15010000004000000001",
		Password:   "123456",
		ClientIP:   "192.168.1.176",
		ClientPort: "5065"}

	//sip.ParseResponseInvite("")
	//return
	/*	bufRTCP := rtcp.GetRR(1, 2, 3)
		for i := 0; i < len(bufRTCP); i++ {
			fmt.Printf("%x-", bufRTCP[i])
		}
		return
	*/
	//testps2es()

	conn, err := net.Dial("udp", uac.ServerIP+":"+uac.ServerPort)
	defer conn.Close()

	if err != nil {
		println("build socket failed")
		return
	}

	go SipEventProc(conn, uac)
	endproc(conn, uac)
}

func endproc(conn net.Conn, uac *sip.UASInfo) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	fmt.Println("Got signal:", s)

	request := uac.BuildBYERequest("34020000001310000001")
	println("-----", request)
	conn.Write([]byte(request))
	/*	buf := make([]byte, 2048)
		length, err := conn.Read(buf)
		if err == nil {
			println(string(buf[:length]))
			errCode, _ := sip.ParseResponseHead(string(buf[:length]))
			if errCode == "100" {

			} else if errCode == "200" {

			}
		} else {
			println("read error ")
		}
	*/
}

func SipEventProc(conn net.Conn, uac *sip.UASInfo) {
	status := REGISTER
	buf := make([]byte, 2048)

	for {
		if status == REGISTER {
			request := uac.BuildRegisterRequest()
			println(request)
			conn.Write([]byte(request))

			length, err := conn.Read(buf)
			if err == nil {
				println(string(buf[:length]))
			} else {
				println("register1 read error")
			}

			errCode, realm, nonce, authMethod, err := sip.ParseRegister1(string(buf[:length]))
			if errCode == "401" && authMethod == "MD5" {
				request := uac.BuildRegisterMD5Auth(realm, nonce)
				println(request)
				conn.Write([]byte(request))
				length, err := conn.Read(buf)
				if err == nil {
					println(string(buf[:length]))
				} else {
					println("register2 read error")
				}

				errCode, err := sip.ParseRegister2(string(buf[:length]))
				if err == nil && errCode == "200" {
					status = LIVEPLAY
				}
			} else {
				println("register1 error", errCode, authMethod)
			}
		} else if status == LIVEPLAY {
			request := uac.BuildInviteRequest(50010, "34020000001310000001")
			println(request)
			conn.Write([]byte(request))

			for {
				length, err := conn.Read(buf)
				if err == nil {
					println(string(buf[:length]))
					errCode, _ := sip.ParseResponseHead(string(buf[:length]))
					if errCode == "100" {
					} else if errCode == "200" {
						remoteMediaPort, remoteSSRC, totag, _ := sip.ParseResponseInvite(string(buf[:length]))
						println("serverport=", remoteMediaPort)
						uac.RemoteMediaPort = remoteMediaPort
						uac.RemoteSSRC = remoteSSRC
						uac.LocalMediaPort = "50010"
						uac.PlayToTag = totag
						break
					}
				} else {
					println("invite read error")
				}
			}

			request = uac.BuildACKRequest("34020000001310000001")
			println(request)
			conn.Write([]byte(request))
			conn.Write([]byte(request))
			go MediaStreamProc(uac)
			length, err := conn.Read(buf)
			if err == nil {
				println(string(buf[:length]))
				errCode, _ := sip.ParseResponseHead(string(buf[:length]))
				if errCode == "100" {
				} else if errCode == "200" {
					status = UNKNOWN
				}
			} else {
				println("ack read error")
			}
		} else if status == UNKNOWN {
			request := uac.BuildHeartbeat()
			println(request)
			conn.Write([]byte(request))

			println("UNKNOWN read packet--------------")
			length, err := conn.Read(buf)
			if err == nil {
				println(string(buf[:length]))
			} else {
				println("UNKNOWN read error")
			}
		}
		time.Sleep(3e9)
	}

}

func MediaStreamProc(uac *sip.UASInfo) {
	udp_addr, err := net.ResolveUDPAddr("udp", ":50010")
	if err != nil {
		println("MediaStreamProc ResolveUDPAddr failed")
		return
	}
	conn, err := net.ListenUDP("udp", udp_addr)
	defer conn.Close()
	if err != nil {
		println("MediaStreamProc ListenUDP failed")
		return
	}

	remotertpport, _ := strconv.Atoi(uac.RemoteMediaPort)
	remotertpport++
	remotertcpport := strconv.Itoa(remotertpport)
	connRTCP, errRTCP := net.Dial("udp", "192.168.6.72:"+remotertcpport)
	defer connRTCP.Close()

	remotessrc, _ := strconv.Atoi(uac.RemoteSSRC)
	localssrc, _ := strconv.Atoi(uac.LocalSSRC)

	println("-----------------", remotessrc, uac.RemoteSSRC, localssrc, uac.LocalSSRC)

	if errRTCP != nil {
		println("build RTCP socket failed")
		return
	}

	file, err := os.OpenFile("111.264", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		println("open file failed.", err.Error())
		return
	}
	defer file.Close()

	//chandata = make(chan []byte, 300*1024)
	//go StreamBufferProc(uac)

	var maxseqnum uint16
	var packetsnum uint
	buf := make([]byte, 2048)

	for {
		length, _, err := conn.ReadFromUDP(buf)
		if err == nil {
			var seqnum uint16
			bufTemp := bytes.NewBuffer(buf[2:])
			binary.Read(bufTemp, binary.BigEndian, &(seqnum))
			if maxseqnum < seqnum {
				maxseqnum = seqnum
			}

			packetsnum++
			println("MediaStreamProc", seqnum, buf[0], buf[1], buf[2], buf[3], length)

			if packetsnum > 5000 {
				bufRTCP := rtcp.GetRR(0x11111111, uint32(remotessrc), maxseqnum)
				connRTCP.Write([]byte(bufRTCP))
				packetsnum = 0
			}

			SetData(buf[12:length], file)
			//			chandata <- buf[12:length]
		} else {
			println("MediaStreamProc read error")
		}

		//		time.Sleep(5e6)
	}
}

func ParsePacket(buffer []byte) (pesbuf []byte, peslen, startpos int) {
	var nFirstPesPos int = -1
	var i int

	for i = 0; i < len(buffer)-4; i++ {
		if (buffer[i]) == (0) &&
			(buffer[i+1]) == (0) &&
			(buffer[i+2]) == (1) &&
			(buffer[i+3]) == (0xe0) {
			nFirstPesPos = i
			i++
			break
		}
	}
	if nFirstPesPos < 0 {
		println("nFirstPesPos < 0 ")
		return nil, 0, -1
	}

	var nPesEndPos int = -1
	for ; i < len(buffer)-4; i++ {
		if buffer[i] == 0 && buffer[i+1] == 0 && buffer[i+2] == 1 &&
			(buffer[i+3] == 0xba || buffer[i+3] == 0xc0 || buffer[i+3] == 0xe0) {
			nPesEndPos = i
			break
		}
	}
	if nPesEndPos < 0 {
		println("nPesEndPos < 0 ")
		return nil, 0, -1
	}

	return buffer[nFirstPesPos:nPesEndPos], nPesEndPos - nFirstPesPos, nFirstPesPos
}

func SetData(data []byte, file *os.File) {
	if offset > len(databuffer) {
		println("databuffer overflow ")
		return
	}

	copy(databuffer[offset:], data)
	offset += len(data)
	if offset < 100*1024 {
		return
	}

	retbuf, peslen, startpos := ParsePacket(databuffer[0:offset])
	for peslen > 0 {
		pesheaderlen := 9 + uint8(retbuf[8])
		buf := retbuf[pesheaderlen:]
		file.Write(buf[0:])

		copy(databuffer[0:], databuffer[startpos+peslen:])
		offset = offset - startpos - peslen
		retbuf, peslen, startpos = ParsePacket(databuffer[0:offset])
	}
}

func testps2es() {
	//	go StreamBufferProc(uac)
	fileps, err := os.OpenFile("111.ps", os.O_RDWR, 0666)
	//	file, err := os.OpenFile("111.ps", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		println("open file failed.", err.Error())
		return
	}
	defer fileps.Close()

	//	go StreamBufferProc(uac)
	file264, err2 := os.OpenFile("111.264", os.O_WRONLY|os.O_CREATE, 0666)
	//	file, err := os.OpenFile("111.ps", os.O_WRONLY|os.O_CREATE, 0666)
	if err2 != nil {
		println("open file failed.", err.Error())
		return
	}
	defer file264.Close()

	buf := make([]byte, 100*1024)
	for true {
		fileps.Read(buf)

		fmt.Printf("%x %x %x %x \n", buf[0], buf[1], buf[2], buf[3])
		SetData(buf, file264)

		time.Sleep(3e8)
	}
}

func StreamBufferProc(uac *sip.UASInfo) {

	file, err := os.OpenFile("111.264", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		println("open file failed.", err.Error())
		return
	}
	defer file.Close()

	for {
		//		file.Write(buf[12:length])

		buf := <-chandata
		println("bufferlen=========", len(buf))

		SetData(buf[0:len(buf)], file)
	}
}
