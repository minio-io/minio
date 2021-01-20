package cmd

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *healingTracker) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ID":
			z.ID, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "ID")
				return
			}
		case "SetIndex":
			z.SetIndex, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "SetIndex")
				return
			}
		case "Path":
			z.Path, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Path")
				return
			}
		case "Endpoint":
			z.Endpoint, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Endpoint")
				return
			}
		case "Started":
			z.Started, err = dc.ReadTime()
			if err != nil {
				err = msgp.WrapError(err, "Started")
				return
			}
		case "LastUpdate":
			z.LastUpdate, err = dc.ReadTime()
			if err != nil {
				err = msgp.WrapError(err, "LastUpdate")
				return
			}
		case "ObjectsHealed":
			z.ObjectsHealed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ObjectsHealed")
				return
			}
		case "ObjectsFailed":
			z.ObjectsFailed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ObjectsFailed")
				return
			}
		case "BytesDone":
			z.BytesDone, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "BytesDone")
				return
			}
		case "BytesFailed":
			z.BytesFailed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "BytesFailed")
				return
			}
		case "Bucket":
			z.Bucket, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Bucket")
				return
			}
		case "Object":
			z.Object, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Object")
				return
			}
		case "ResumeObjectsHealed":
			z.ResumeObjectsHealed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ResumeObjectsHealed")
				return
			}
		case "ResumeObjectsFailed":
			z.ResumeObjectsFailed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ResumeObjectsFailed")
				return
			}
		case "ResumeBytesDone":
			z.ResumeBytesDone, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ResumeBytesDone")
				return
			}
		case "ResumeBytesFailed":
			z.ResumeBytesFailed, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "ResumeBytesFailed")
				return
			}
		case "QueuedBuckets":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "QueuedBuckets")
				return
			}
			if cap(z.QueuedBuckets) >= int(zb0002) {
				z.QueuedBuckets = (z.QueuedBuckets)[:zb0002]
			} else {
				z.QueuedBuckets = make([]string, zb0002)
			}
			for za0001 := range z.QueuedBuckets {
				z.QueuedBuckets[za0001], err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "QueuedBuckets", za0001)
					return
				}
			}
		case "HealedBuckets":
			var zb0003 uint32
			zb0003, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "HealedBuckets")
				return
			}
			if cap(z.HealedBuckets) >= int(zb0003) {
				z.HealedBuckets = (z.HealedBuckets)[:zb0003]
			} else {
				z.HealedBuckets = make([]string, zb0003)
			}
			for za0002 := range z.HealedBuckets {
				z.HealedBuckets[za0002], err = dc.ReadString()
				if err != nil {
					err = msgp.WrapError(err, "HealedBuckets", za0002)
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *healingTracker) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 18
	// write "ID"
	err = en.Append(0xde, 0x0, 0x12, 0xa2, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteString(z.ID)
	if err != nil {
		err = msgp.WrapError(err, "ID")
		return
	}
	// write "SetIndex"
	err = en.Append(0xa8, 0x53, 0x65, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78)
	if err != nil {
		return
	}
	err = en.WriteInt(z.SetIndex)
	if err != nil {
		err = msgp.WrapError(err, "SetIndex")
		return
	}
	// write "Path"
	err = en.Append(0xa4, 0x50, 0x61, 0x74, 0x68)
	if err != nil {
		return
	}
	err = en.WriteString(z.Path)
	if err != nil {
		err = msgp.WrapError(err, "Path")
		return
	}
	// write "Endpoint"
	err = en.Append(0xa8, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return
	}
	err = en.WriteString(z.Endpoint)
	if err != nil {
		err = msgp.WrapError(err, "Endpoint")
		return
	}
	// write "Started"
	err = en.Append(0xa7, 0x53, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteTime(z.Started)
	if err != nil {
		err = msgp.WrapError(err, "Started")
		return
	}
	// write "LastUpdate"
	err = en.Append(0xaa, 0x4c, 0x61, 0x73, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65)
	if err != nil {
		return
	}
	err = en.WriteTime(z.LastUpdate)
	if err != nil {
		err = msgp.WrapError(err, "LastUpdate")
		return
	}
	// write "ObjectsHealed"
	err = en.Append(0xad, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ObjectsHealed)
	if err != nil {
		err = msgp.WrapError(err, "ObjectsHealed")
		return
	}
	// write "ObjectsFailed"
	err = en.Append(0xad, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ObjectsFailed)
	if err != nil {
		err = msgp.WrapError(err, "ObjectsFailed")
		return
	}
	// write "BytesDone"
	err = en.Append(0xa9, 0x42, 0x79, 0x74, 0x65, 0x73, 0x44, 0x6f, 0x6e, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.BytesDone)
	if err != nil {
		err = msgp.WrapError(err, "BytesDone")
		return
	}
	// write "BytesFailed"
	err = en.Append(0xab, 0x42, 0x79, 0x74, 0x65, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.BytesFailed)
	if err != nil {
		err = msgp.WrapError(err, "BytesFailed")
		return
	}
	// write "Bucket"
	err = en.Append(0xa6, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74)
	if err != nil {
		return
	}
	err = en.WriteString(z.Bucket)
	if err != nil {
		err = msgp.WrapError(err, "Bucket")
		return
	}
	// write "Object"
	err = en.Append(0xa6, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74)
	if err != nil {
		return
	}
	err = en.WriteString(z.Object)
	if err != nil {
		err = msgp.WrapError(err, "Object")
		return
	}
	// write "ResumeObjectsHealed"
	err = en.Append(0xb3, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ResumeObjectsHealed)
	if err != nil {
		err = msgp.WrapError(err, "ResumeObjectsHealed")
		return
	}
	// write "ResumeObjectsFailed"
	err = en.Append(0xb3, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ResumeObjectsFailed)
	if err != nil {
		err = msgp.WrapError(err, "ResumeObjectsFailed")
		return
	}
	// write "ResumeBytesDone"
	err = en.Append(0xaf, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x42, 0x79, 0x74, 0x65, 0x73, 0x44, 0x6f, 0x6e, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ResumeBytesDone)
	if err != nil {
		err = msgp.WrapError(err, "ResumeBytesDone")
		return
	}
	// write "ResumeBytesFailed"
	err = en.Append(0xb1, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x42, 0x79, 0x74, 0x65, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.ResumeBytesFailed)
	if err != nil {
		err = msgp.WrapError(err, "ResumeBytesFailed")
		return
	}
	// write "QueuedBuckets"
	err = en.Append(0xad, 0x51, 0x75, 0x65, 0x75, 0x65, 0x64, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.QueuedBuckets)))
	if err != nil {
		err = msgp.WrapError(err, "QueuedBuckets")
		return
	}
	for za0001 := range z.QueuedBuckets {
		err = en.WriteString(z.QueuedBuckets[za0001])
		if err != nil {
			err = msgp.WrapError(err, "QueuedBuckets", za0001)
			return
		}
	}
	// write "HealedBuckets"
	err = en.Append(0xad, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.HealedBuckets)))
	if err != nil {
		err = msgp.WrapError(err, "HealedBuckets")
		return
	}
	for za0002 := range z.HealedBuckets {
		err = en.WriteString(z.HealedBuckets[za0002])
		if err != nil {
			err = msgp.WrapError(err, "HealedBuckets", za0002)
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *healingTracker) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 18
	// string "ID"
	o = append(o, 0xde, 0x0, 0x12, 0xa2, 0x49, 0x44)
	o = msgp.AppendString(o, z.ID)
	// string "SetIndex"
	o = append(o, 0xa8, 0x53, 0x65, 0x74, 0x49, 0x6e, 0x64, 0x65, 0x78)
	o = msgp.AppendInt(o, z.SetIndex)
	// string "Path"
	o = append(o, 0xa4, 0x50, 0x61, 0x74, 0x68)
	o = msgp.AppendString(o, z.Path)
	// string "Endpoint"
	o = append(o, 0xa8, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendString(o, z.Endpoint)
	// string "Started"
	o = append(o, 0xa7, 0x53, 0x74, 0x61, 0x72, 0x74, 0x65, 0x64)
	o = msgp.AppendTime(o, z.Started)
	// string "LastUpdate"
	o = append(o, 0xaa, 0x4c, 0x61, 0x73, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65)
	o = msgp.AppendTime(o, z.LastUpdate)
	// string "ObjectsHealed"
	o = append(o, 0xad, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.ObjectsHealed)
	// string "ObjectsFailed"
	o = append(o, 0xad, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.ObjectsFailed)
	// string "BytesDone"
	o = append(o, 0xa9, 0x42, 0x79, 0x74, 0x65, 0x73, 0x44, 0x6f, 0x6e, 0x65)
	o = msgp.AppendUint64(o, z.BytesDone)
	// string "BytesFailed"
	o = append(o, 0xab, 0x42, 0x79, 0x74, 0x65, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.BytesFailed)
	// string "Bucket"
	o = append(o, 0xa6, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74)
	o = msgp.AppendString(o, z.Bucket)
	// string "Object"
	o = append(o, 0xa6, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74)
	o = msgp.AppendString(o, z.Object)
	// string "ResumeObjectsHealed"
	o = append(o, 0xb3, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.ResumeObjectsHealed)
	// string "ResumeObjectsFailed"
	o = append(o, 0xb3, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.ResumeObjectsFailed)
	// string "ResumeBytesDone"
	o = append(o, 0xaf, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x42, 0x79, 0x74, 0x65, 0x73, 0x44, 0x6f, 0x6e, 0x65)
	o = msgp.AppendUint64(o, z.ResumeBytesDone)
	// string "ResumeBytesFailed"
	o = append(o, 0xb1, 0x52, 0x65, 0x73, 0x75, 0x6d, 0x65, 0x42, 0x79, 0x74, 0x65, 0x73, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64)
	o = msgp.AppendUint64(o, z.ResumeBytesFailed)
	// string "QueuedBuckets"
	o = append(o, 0xad, 0x51, 0x75, 0x65, 0x75, 0x65, 0x64, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.QueuedBuckets)))
	for za0001 := range z.QueuedBuckets {
		o = msgp.AppendString(o, z.QueuedBuckets[za0001])
	}
	// string "HealedBuckets"
	o = append(o, 0xad, 0x48, 0x65, 0x61, 0x6c, 0x65, 0x64, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.HealedBuckets)))
	for za0002 := range z.HealedBuckets {
		o = msgp.AppendString(o, z.HealedBuckets[za0002])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *healingTracker) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ID":
			z.ID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ID")
				return
			}
		case "SetIndex":
			z.SetIndex, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SetIndex")
				return
			}
		case "Path":
			z.Path, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Path")
				return
			}
		case "Endpoint":
			z.Endpoint, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Endpoint")
				return
			}
		case "Started":
			z.Started, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Started")
				return
			}
		case "LastUpdate":
			z.LastUpdate, bts, err = msgp.ReadTimeBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "LastUpdate")
				return
			}
		case "ObjectsHealed":
			z.ObjectsHealed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ObjectsHealed")
				return
			}
		case "ObjectsFailed":
			z.ObjectsFailed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ObjectsFailed")
				return
			}
		case "BytesDone":
			z.BytesDone, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BytesDone")
				return
			}
		case "BytesFailed":
			z.BytesFailed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BytesFailed")
				return
			}
		case "Bucket":
			z.Bucket, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Bucket")
				return
			}
		case "Object":
			z.Object, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Object")
				return
			}
		case "ResumeObjectsHealed":
			z.ResumeObjectsHealed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ResumeObjectsHealed")
				return
			}
		case "ResumeObjectsFailed":
			z.ResumeObjectsFailed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ResumeObjectsFailed")
				return
			}
		case "ResumeBytesDone":
			z.ResumeBytesDone, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ResumeBytesDone")
				return
			}
		case "ResumeBytesFailed":
			z.ResumeBytesFailed, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ResumeBytesFailed")
				return
			}
		case "QueuedBuckets":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "QueuedBuckets")
				return
			}
			if cap(z.QueuedBuckets) >= int(zb0002) {
				z.QueuedBuckets = (z.QueuedBuckets)[:zb0002]
			} else {
				z.QueuedBuckets = make([]string, zb0002)
			}
			for za0001 := range z.QueuedBuckets {
				z.QueuedBuckets[za0001], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "QueuedBuckets", za0001)
					return
				}
			}
		case "HealedBuckets":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "HealedBuckets")
				return
			}
			if cap(z.HealedBuckets) >= int(zb0003) {
				z.HealedBuckets = (z.HealedBuckets)[:zb0003]
			} else {
				z.HealedBuckets = make([]string, zb0003)
			}
			for za0002 := range z.HealedBuckets {
				z.HealedBuckets[za0002], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "HealedBuckets", za0002)
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *healingTracker) Msgsize() (s int) {
	s = 3 + 3 + msgp.StringPrefixSize + len(z.ID) + 9 + msgp.IntSize + 5 + msgp.StringPrefixSize + len(z.Path) + 9 + msgp.StringPrefixSize + len(z.Endpoint) + 8 + msgp.TimeSize + 11 + msgp.TimeSize + 14 + msgp.Uint64Size + 14 + msgp.Uint64Size + 10 + msgp.Uint64Size + 12 + msgp.Uint64Size + 7 + msgp.StringPrefixSize + len(z.Bucket) + 7 + msgp.StringPrefixSize + len(z.Object) + 20 + msgp.Uint64Size + 20 + msgp.Uint64Size + 16 + msgp.Uint64Size + 18 + msgp.Uint64Size + 14 + msgp.ArrayHeaderSize
	for za0001 := range z.QueuedBuckets {
		s += msgp.StringPrefixSize + len(z.QueuedBuckets[za0001])
	}
	s += 14 + msgp.ArrayHeaderSize
	for za0002 := range z.HealedBuckets {
		s += msgp.StringPrefixSize + len(z.HealedBuckets[za0002])
	}
	return
}
