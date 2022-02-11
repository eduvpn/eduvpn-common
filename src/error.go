package eduvpn

type detailedVPNErrorCode int8
type VPNErrorCode int8

type VPNError struct {
	Code     VPNErrorCode
	Detailed detailedVPNError
}

func (err VPNError) Error() string {
	return err.Detailed.Error()
}

func (err VPNError) Unwrap() error {
	return err.Detailed
}

type detailedVPNError struct {
	Code    detailedVPNErrorCode
	Message string
	Cause   error
}

func (err detailedVPNError) Error() string {
	return err.Message
}
func (err detailedVPNError) Unwrap() error {
	return err.Cause
}
