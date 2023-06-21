package crc16

const polynomial = 0xA001

func Checksum(data []byte) uint16 {
	crc := uint16(0x0000)
	for _, b := range data {
		crc = crc ^ uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) > 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc = (crc >> 1)
			}
		}
	}
	return crc
}
