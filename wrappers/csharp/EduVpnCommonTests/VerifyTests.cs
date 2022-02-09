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
			Discovery.InsecureTestingSetExtraKey(File.ReadLines($"{testDataDir_}/public.key").Last());

		[Test]
		[TestCase("server_list.json.minisig",       "server_list.json",       "server_list.json")]
		[TestCase("organization_list.json.minisig", "organization_list.json", "organization_list.json")]
		public void TestValid(string sigFile, string jsonFile, string expectedFileName) =>
			Discovery.Verify(
				File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
				File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
				expectedFileName,
				DateTimeOffset.FromUnixTimeSeconds(10));

		[Test]
		[TestCase("server_list.json.minisig", "server_list.json", "server_list.json")]
		public void TestValidSegment(string sigFile, string jsonFile, string expectedFileName)
		{
			var bytes = new byte[] { 1, 2, 3 }.Concat(File.ReadAllBytes($"{testDataDir_}/{jsonFile}"))
				.Concat(new byte[] { 1, 2, 3 }).ToArray();
			Discovery.Verify(
				File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
				new(bytes, 3, bytes.Length - 3 - 3),
				expectedFileName,
				DateTimeOffset.UnixEpoch);
		}

		[Test]
		[TestCase("random.txt", "server_list.json", "server_list.json")]
		public void TestInvalidSignature(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws<InvalidSignatureException>(
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));

		[Test]
		[TestCase("server_list.json.wrong_key.minisig", "server_list.json", "server_list.json")]
		public void TestWrongKey(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws<InvalidSignatureUnknownKeyException>(
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));

		[Test]
		[TestCase("server_list.json.minisig", "server_list.json", "server_list.json")]
		public void TestOldSignature(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws<SignatureTooOldException>(
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.FromUnixTimeSeconds(11)));

		[Test]
		[TestCase("other_list.json.minisig", "other_list.json", "other_list.json")]
		public void TestUnknownExpectedFile(string sigFile, string jsonFile, string expectedFileName) =>
			Assert.Throws<ArgumentException>(
				() => Discovery.Verify(
					File.ReadAllBytes($"{testDataDir_}/{sigFile}"),
					File.ReadAllBytes($"{testDataDir_}/{jsonFile}"),
					expectedFileName,
					DateTimeOffset.UnixEpoch));
	}
}
