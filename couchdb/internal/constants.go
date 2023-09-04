// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

// Package internal contains some internal constants.
package internal

// Common constants, placed here to allow importing in chttp and root package
// without import cycles.
const (
	OptionUserAgent            = "User-Agent"
	OptionHTTPClient           = "kivik:httpClient"
	OptionFullCommit           = "X-Couch-Full-Commit"
	OptionIfNoneMatch          = "If-None-Match"
	OptionPartition            = "kivik:partition"
	OptionNoMultipartPut       = "kivik:no-multipart-put"
	OptionNoMultipartGet       = "kivik:no-multipart-get"
	OptionNoCompressedRequests = "kivik:no-compressed-requests"
)