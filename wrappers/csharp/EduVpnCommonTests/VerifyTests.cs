using System;
using System.IO;
using System.Linq;
using EduVpnCommon;
using NUnit.Framework;

namespace EduVpnCommonTests
{
	[TestFixture(TestOf = typeof(Discovery)), Parallelizable]
	public class VerifyTests
	{
		// Relative to e.g. EduVpnCommonTests/bin/Debug/net5.0
		readonly string testDataDir_ = $"{TestContext.CurrentContext.TestDirectory}/../../../../../../test_data";

		[OneTimeSetUp]
		public void OneTimeSetUp() =>
			Discovery.InsecureTestingSetExtraKey(File.ReadLines($"{testDataDir_}/dummy/public.key").Last());

		[Test]
		[TestCase("dummy/server_list.json.minisig",       "dummy/server_list.json",       "server_list.json")]
		[TestCase("dummy/organization_list.json.minisig", "dummy/organization_list.json", "organization_list.json")]
		public void TestValid(string sigFile, string jsonFile, string expectedFileName) =>
			Discovery.Verify(
				File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
				File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
				expectedFileName,
				DateTimeOffset.UnixEpoch);
		
		[Test]
		[TestCase("dummy/random.txt", "dummy/server_list.json", "server_list.json")]
		public void TestInvalidSignature(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws(Is.TypeOf<VerifyException>()
					.And.Property(nameof(VerifyException.Code)).EqualTo(VerifyErrorCode.ErrInvalidSignature),
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));

		[Test]
		[TestCase("dummy/server_list.json.wrong_key.minisig", "dummy/server_list.json", "server_list.json")]
		public void TestWrongKey(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws(Is.TypeOf<VerifyException>()
					.And.Property(nameof(VerifyException.Code)).EqualTo(VerifyErrorCode.ErrInvalidSignatureUnknownKey),
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));

		[Test]
		[TestCase("dummy/server_list.json.minisig", "dummy/server_list.json", "server_list.json")]
		public void TestOldSignature(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws(Is.TypeOf<VerifyException>()
					.And.Property(nameof(VerifyException.Code)).EqualTo(VerifyErrorCode.ErrTooOld),
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.MaxValue));

		[Test]
		[TestCase("dummy/other_list.json.minisig", "dummy/other_list.json", "other_list.json")]
		public void TestUnknownExpectedFile(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws<ArgumentException>(
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));
	}
}