package main 

import (
	"log"
	"flag"
	"trip_time/msg"
	"net"
	"time"
	"github.com/golang/protobuf/proto"
	//"bufio"
)


func exactSleep (duration time.Duration) {
	s := time.Now()
	d := duration * 9 / 10
	oneMs := time.Millisecond
	tenMs := time.Millisecond*10
	for {
		time.Sleep(d)
		if duration < time.Since(s) {
			break
		}
		left := duration - time.Since(s)
		if oneMs < left && tenMs > left {
			d = oneMs
		} else {
			d = left * 10 / 9
		}
	}
}

func disp_tick (start int64) {
	var lastSecond int64 = 0
	for {
		d := (time.Now().UnixNano() - start)/(1000*1000*1000)
		if d > lastSecond {
			log.Printf("Tick = %ds\n", d)
			lastSecond = d
		}
		exactSleep (time.Millisecond*10)
	}
}

///接收到ReqTT, 
func tt_server(port int) {
	start := time.Now().UnixNano()
	go disp_tick (start)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if nil != err {
		log.Panic(err)
	}

	buffer := make([]byte, 1500)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Panic(err)
		}
		req := &msg.ReqTT {}
		if err := proto.Unmarshal (buffer[:n], req); nil != err {
			log.Panic(err)
		}
		log.Println ("Recv", req, "From", clientAddr)
		//每50ms发送一个TTPack, 连续发送20个
		i := 0
		wd := time.Now()
		for i < 20 {
			wwd := time.Now()
			resp := &msg.TTPack {
				Index : int32(i),
				TimeStamp : time.Now().UnixNano() - start,
			}
			out,err := proto.Marshal (resp)
			if nil != err {
				log.Panic(err)
			}
			conn.WriteTo (out, clientAddr)
			exactSleep (time.Millisecond * 50 - time.Since(wwd))
			i += 1
		}
		log.Println(time.Since(wd))
	}
}



func tt_client(server string, port int) {
	conn, err := net.DialUDP("udp", 
								&net.UDPAddr{IP: net.IPv4zero, Port: 0}, 
								&net.UDPAddr{IP: net.ParseIP(server), 
								Port: int(port)})
	if err != nil {
		log.Panic(err)
	}
	req := &msg.ReqTT{1024}
	out,err := proto.Marshal (req)
	if nil != err {
		log.Panic(err)
	}
	conn.Write (out)

	///接收一次TTPack,更新TT
	buffer := make([]byte, 1500)
	first := true
	var firstTs int64
	var ts int64
	var idx int32
	var nTTPack int64 = 0
	var accInterval int64 = 0
	var tick int64
	var tickDiff int64
	for {
		conn.SetReadDeadline (time.Now ().Add(time.Second))
		n, err := conn.Read (buffer)
		if nil != err {
			break
		}
		resp := &msg.TTPack{}
		if err := proto.Unmarshal (buffer[:n], resp); nil != err {
			log.Panic(err)
		}
		log.Println(resp)

		if first {
			firstTs = resp.TimeStamp
			idx = resp.Index
			ts  = resp.TimeStamp
			nTTPack += 1
			first = false
			log.Printf("TimeStamp = %d, Index = %d\n", firstTs, idx)
		} else {
			if resp.Index == idx + 1 {
				nTTPack += 1
				idx = resp.Index
				accInterval += (resp.TimeStamp - ts)
				avgInterval := accInterval/(nTTPack-1)
				ts  = resp.TimeStamp
				tick = ts + avgInterval
				tickDiff = time.Now().UnixNano()
				log.Printf("TimeStamp = %d, Index = %d, interval = %dms\n", ts, idx, avgInterval/1000/1000)
			}
		}
	}

	disp_tick (tickDiff - tick)
}



func main() {
	role := flag.String("role", "server", "server/client")
	server := flag.String("server", "127.0.0.1", "tt server ip")
	port := flag.Int("port", 1888, "tt server listen port")
	flag.Parse()
	if "client" == *role {
		tt_client(*server, *port)
	} else {
		tt_server(*port)
	}
}
