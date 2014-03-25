// Copyright (c) 2013,2014 SmugMug, Inc. All rights reserved.
// 
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
//       copyright notice, this list of conditions and the following
//       disclaimer in the documentation and/or other materials provided
//       with the distribution.
// 
// THIS SOFTWARE IS PROVIDED BY SMUGMUG, INC. ``AS IS'' AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
// PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL SMUGMUG, INC. BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE
// GOODS OR SERVICES;LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER
// IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR
// OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF
// ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Manages signing tasks required by AWS Auth v4 requests to DynamoDB.
// See http://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
// for more information on v4 signed requests.
package tasks

import (
	"strings"
	"hash"
	"time"
	"crypto/sha256"
	"crypto/hmac"
	"encoding/hex"
	"github.com/smugmug/godynamo/aws_const"
)

// MakeSignature returns a auth_v4 signature from the `string to sign` variable.
// May be useful for creating v4 requests for services other than DynamoDB.
func MakeSignature(string2sign,zone,service,secret string) string {
	kCredentials,_ := cacheable_hmacs(zone,service,secret)
	var kSigning_hmac_sha256 hash.Hash = hmac.New(sha256.New,kCredentials)
        kSigning_hmac_sha256.Write([]byte(string2sign))
        kSigning := kSigning_hmac_sha256.Sum(nil)
	return hex.EncodeToString(kSigning)
}

// Return the byte slice for the cacheable hmac, along with the date string
// that describes its time of creation.
func cacheable_hmacs(zone,service,secret string) ([]byte,string) {
	t := time.Now().UTC()
	gmt_yyyymmdd := t.Format(aws_const.ISODATEFMT)

	init_secret := []byte("AWS4" + secret)
        var kDate_hmac_sha256 hash.Hash = hmac.New(sha256.New,init_secret)
        kDate_hmac_sha256.Write([]byte(gmt_yyyymmdd))
        kDate := kDate_hmac_sha256.Sum(nil)

	var kRegion_hmac_sha256 hash.Hash = hmac.New(sha256.New,kDate)
        kRegion_hmac_sha256.Write([]byte(zone))
        kRegion := kRegion_hmac_sha256.Sum(nil)

	var kService_hmac_sha256 hash.Hash = hmac.New(sha256.New,kRegion)
        kService_hmac_sha256.Write([]byte(service))
        kService := kService_hmac_sha256.Sum(nil)

	var kCredentials_hmac_sha256 hash.Hash =
		hmac.New(sha256.New,kService)
        kCredentials_hmac_sha256.Write([]byte("aws4_request"))
        return kCredentials_hmac_sha256.Sum(nil),gmt_yyyymmdd
}

// CanonicalRequest will create the aws v4 `canonical request`.
// May be useful for creating v4 requests for services other than DynamoDB.
func CanonicalRequest(host,port,amzDateHdr,amzTargetHdr,hexPayload string) string {
	// Some AWS services use the x-amz-target header. Some don't. Allow it to
	// be passed as empty when not used.
	amzTarget_list_elt := ""
	amzTarget_val      := ""
	if amzTargetHdr != "" {
		amzTarget_val = strings.ToLower(aws_const.AMZ_TARGET_HDR) + ":" +
			amzTargetHdr + "\n"
		amzTarget_list_elt = ";x-amz-target"
	}
	return aws_const.METHOD + "\n" +
	 	"/" + "\n" +
	 	"" + "\n" +
		strings.ToLower(aws_const.CONTENT_TYPE_HDR) + ":" +
		aws_const.CTYPE + "\n" +
		strings.ToLower("host") + ":" +
		host + ":" +
		port + "\n" +
		strings.ToLower(aws_const.X_AMZ_DATE_HDR) + ":" +
		amzDateHdr + "\n" +
		amzTarget_val +
		"\n" +
		"content-type;host;x-amz-date" + amzTarget_list_elt + "\n" +
		hexPayload
}

// String2Sign will create the aws v4 `string to sign` from the `canoncial request`.
// May be useful for creating v4 requests for services other than DynamoDB.
func String2Sign(t time.Time,canonical_request,zone,service string) string {
	var h256 hash.Hash = sha256.New()
	h256.Write([]byte(canonical_request))
	hexCanonReq := hex.EncodeToString([]byte(h256.Sum(nil)))
	return "AWS4-HMAC-SHA256" + "\n" +
		t.UTC().Format(aws_const.ISO8601FMT_CONDENSED) + "\n" +
		t.UTC().Format(aws_const.ISODATEFMT) + "/" +
		zone +
		"/" + service + "/aws4_request" + "\n" +
		hexCanonReq
}
