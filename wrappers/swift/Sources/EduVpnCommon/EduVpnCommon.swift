import Foundation
import CEduVpnCommon

private extension Data {
    /// Execute `body` with `GoSlice` pointing to `self`
    /// - Important: `GoSlice` pointer must not be written to
    func withSlice<ResultType>(_ body: (GoSlice) throws -> ResultType) rethrows -> ResultType {
        // Could also use raw NSData.bytes, but then you have to be careful that the NSData must remain valid during the call
        // This closure method guarantees this
        try withUnsafeBytes { (pointer: UnsafeRawBufferPointer) -> ResultType in
            // Note: UnsafeRawBufferPointer.startIndex will always be 0, see docs
            // Cast to UnsafeMutableRawPointer, assumes it will not be written to
            try body(GoSlice(data: UnsafeMutableRawPointer(mutating: pointer.baseAddress),
                    len: GoInt(pointer.count), cap: GoInt(pointer.count)))
        }
    }
}

private extension String {
    /// Execute `body` with `GoSlice` pointing to UTF-8 bytes copied from `self`
    func withSlice<ResultType>(_ body: (GoSlice) throws -> ResultType) rethrows -> ResultType {
        try data(using: .utf8)!.withSlice(body)
    }
}

public enum VerifyErr: Error, Equatable {
    /// Expected file name is not one of the recognized values.
    case ErrUnknownExpectedFileName
    /// Signature is invalid (for the expected file type).
    case ErrInvalidSignature
    /// Signature was created with an unknown key and has not been verified.
    case ErrInvalidSignatureUnknownKey
    /// Signature has a timestamp lower than the specified minimum signing time.
    case ErrTooOld
    /// Other unknown error
    case Unknown(code: GoInt)

    static func fromCode(_ code: GoInt) -> VerifyErr {
        precondition(code != 0)
        switch code {
        case 1: return ErrUnknownExpectedFileName
        case 2: return ErrInvalidSignature
        case 3: return ErrInvalidSignatureUnknownKey
        case 4: return ErrTooOld
        default: return Unknown(code: code)
        }
    }
}

/// Verifies the signature on the JSON server_list.json/organization_list.json file.
/// If the function returns, the signature is valid for the given file type.
///
/// - Parameters:
///   - signature: .minisig signature file contents.
///   - signedJson: Signed .json file contents.
///   - expectedFileName: The file type to be verified, one of "server_list.json" or "organization_list.json".
///   - minSignTime: Minimum time for signature. Should be set to at least the time in a previously retrieved file.
/// - Throws: VerifyErr: If signature verification fails or `expectedFileName` is not one of the allowed values.
public func Verify(signature: Data, signedJson: Data, expectedFileName: String, minSignTime: Date) throws {
    let result = signature.withSlice { signatureData in
        signedJson.withSlice { jsonData in
            expectedFileName.withSlice { expectedNameData in
                CEduVpnCommon.Verify(signatureData, jsonData, expectedNameData, GoUint64(minSignTime.timeIntervalSince1970))
            }
        }
    }
    if result != 0 {
        throw VerifyErr.fromCode(result)
    }
}

/// Use for testing only, see Go documentation.
internal func InsecureTestingSetExtraKey(keyString: String) {
    keyString.withSlice { keyData in
        CEduVpnCommon.InsecureTestingSetExtraKey(keyData);
    }
}
