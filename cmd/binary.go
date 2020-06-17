package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"time"
)

type SensorReading struct {
	SensorID   uint16
	LocationID uint16
	Timestamp  int64
	Temp       float64
}

func newSensorReading(sID, locID uint16, temp float64) *SensorReading {
	return &SensorReading{
		SensorID:   sID,
		LocationID: locID,
		Timestamp:  time.Now().Unix(),
		Temp:       temp,
	}
}

func encode(s *SensorReading) []byte {
	buf := make([]byte, 20)
	binary.BigEndian.PutUint16(buf[0:], s.SensorID)
	binary.BigEndian.PutUint16(buf[2:], s.LocationID)
	binary.BigEndian.PutUint64(buf[4:], uint64(s.Timestamp))
	binary.BigEndian.PutUint64(buf[12:], math.Float64bits(s.Temp))
	return buf
}

func decode(buf []byte) *SensorReading {
	s := new(SensorReading)
	s.SensorID = binary.BigEndian.Uint16(buf[0:])
	s.LocationID = binary.BigEndian.Uint16(buf[2:])
	s.Timestamp = int64(binary.BigEndian.Uint64(buf[4:]))
	s.Temp = math.Float64frombits(binary.BigEndian.Uint64(buf[12:]))
	return s
}

func v1() {
	s1 := newSensorReading(70, 1, 33.33)
	buf := encode(s1)
	s2 := decode(buf)
	if !reflect.DeepEqual(s1, s2) {
		fmt.Println("v1: s1 & s2 not equal")
	} else {
		fmt.Println("ok v1")
	}
}

func v2() {
	s1 := newSensorReading(70, 1, 33.33)
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, s1)
	if err != nil {
		panic(err)
	}
	s2 := new(SensorReading)
	err = binary.Read(buf, binary.BigEndian, s2)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(s1, s2) {
		fmt.Println("v2: s1 & s2 not equal")
	} else {
		fmt.Println("ok v2")
	}
}
