package utils

import "testing"

func TestNewConvertionData(t *testing.T) {
	unit_family, rate, unit_index := Get_supported_unit_family_and_convertion_rate("time", "millisecond")
	data, unit_name, unit_short := Data_convertion(unit_family, rate, 2000, unit_index)
	println(data, unit_name, unit_short)
}
