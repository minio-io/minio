// Code generated by "stringer -type=Op -output=msg_string.go -trimprefix=Op msg.go"; DO NOT EDIT.

package wsconn

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OpConnect-1]
	_ = x[OpRequest-2]
	_ = x[OpResponse-3]
	_ = x[OpConnectMux-4]
	_ = x[OpDisconnectMux-5]
	_ = x[OpMsg-6]
	_ = x[OpUnblock-7]
	_ = x[OpError-8]
	_ = x[OpDisconnect-9]
}

const _Op_name = "ConnectRequestResponseConnectMuxDisconnectMuxMsgUnblockErrorDisconnect"

var _Op_index = [...]uint8{0, 7, 14, 22, 32, 45, 48, 55, 60, 70}

func (i Op) String() string {
	i -= 1
	if i >= Op(len(_Op_index)-1) {
		return "Op(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Op_name[_Op_index[i]:_Op_index[i+1]]
}
