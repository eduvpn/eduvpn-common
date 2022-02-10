import XCTest
@testable import EduVpnCommon

final class EduVpnCommonTests: XCTestCase {
    private static let testDataDir = "../../src/test_data"

    override class func setUp() {
        // Swift is confused by CRLF, so on some systems we cannot just take the second-to-last element
        InsecureTestingSetExtraKey(keyString: try! String(contentsOfFile: "\(testDataDir)/public.key")
                .components(separatedBy: .newlines).last(where: { !$0.isEmpty })!)
    }

    func testValid() throws {
        try Verify(
                signature: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json.minisig")),
                signedJson: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json")),
                expectedFileName: "server_list.json",
                minSignTime: Date(timeIntervalSince1970: 10))
    }

    func testInvalidSignature() throws {
        XCTAssertThrowsError(
            try Verify(
                    signature: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/random.txt")),
                    signedJson: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json")),
                    expectedFileName: "server_list.json",
                    minSignTime: Date(timeIntervalSince1970: 0)),
            "", {err in XCTAssertEqual(err as? VerifyErr, VerifyErr.ErrInvalidSignature)});
    }

    func testWrongKey() throws {
        XCTAssertThrowsError(
                try Verify(
                        signature: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json.wrong_key.minisig")),
                        signedJson: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json")),
                        expectedFileName: "server_list.json",
                        minSignTime: Date(timeIntervalSince1970: 0)),
                "", {err in XCTAssertEqual(err as? VerifyErr, VerifyErr.ErrInvalidSignatureUnknownKey)});
    }

    func testOldSignature() throws {
        XCTAssertThrowsError(
                try Verify(
                        signature: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json.minisig")),
                        signedJson: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/server_list.json")),
                        expectedFileName: "server_list.json",
                        minSignTime: Date(timeIntervalSince1970: 11)),
                "", {err in XCTAssertEqual(err as? VerifyErr, VerifyErr.ErrTooOld)});
    }

    func testUnknownExpectedFile() throws {
        XCTAssertThrowsError(
                try Verify(
                        signature: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/other_list.json.minisig")),
                        signedJson: try! Data(contentsOf: URL(fileURLWithPath: "\(EduVpnCommonTests.testDataDir)/other_list.json")),
                        expectedFileName: "other_list.json",
                        minSignTime: Date(timeIntervalSince1970: 0)),
                "", {err in XCTAssertEqual(err as? VerifyErr, VerifyErr.ErrUnknownExpectedFileName)});
    }
}
