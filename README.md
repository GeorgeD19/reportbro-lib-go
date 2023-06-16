# reportbro-lib-go

ReportBro Lib Go is the un-official library to generate PDF and XLSX reports. Report templates can be created 
with `ReportBro Designer <https://github.com/jobsta/reportbro-designer>`,
a Javascript Plugin which can be integrated in your web application.

See the ReportBro project website on https://www.reportbro.com for full documentation and demos.

## Table of contents

- [reportbro-lib-go](#reportbro-lib-go)
  - [Table of contents](#table-of-contents)
  - [1. Overview](#1-overview)
  - [2. Quick Start](#2-quick-start)
  - [3. Prerequisites](#3-prerequisites)
  - [4. Install](#4-install)
  - [5. Running](#5-running)
  - [6. Contributing](#6-contributing)
  - [7. Reporting Issues](#7-reporting-issues)
  - [5. License](#5-license)

## 1. Overview

* Generate pdf and xlsx reports
* Supports (repeating) header and footer
* Allows predefined and own page formats
* Use text, line, images, barcodes and tables, page breaks
* Text and element styling
* Evaluate expressions, define conditional styles, format parameters

## 2. Quick Start

Install and import into your project.


```
go get github.com/GeorgeD19/reportbro-lib-go
```

```
import (
    "github.com/GeorgeD19/reportbro-lib-go"
)
```

## 3. Prerequisites

- [Golang](https://golang.org/)
- [GVM](https://github.com/moovweb/gvm)
- [Dep](https://golang.github.io/dep/)

## 4. Install

Install go dependencies

```
go mod tidy
```

Install node packages for app example
```
cd app && npm install
```

## 5. Running

Start go server

```
cd app && go run server.go
```

Then open index.html in your browser.

## 6. Contributing

Read our [Contribution Guidelines](https://github.com/GeorgeD19/blob/master/CONTRIBUTING.md) for information on how you can help out ReportBro Lib Go.

## 7. Reporting Issues

If you think you've found a bug, or something isn't behaving the way you think it should, please raise an [issue](https://github.com/GeorgeD19/reportbro-lib-go/issues) on Github.

## 5. License

- Commercial license

If you want to use ReportBro to develop commercial applications and projects, the Commercial license is the appropriate license. With this license, your source code is kept proprietary. Purchase a ReportBro Commercial license at https://www.reportbro.com/buy.

- Open-source license

If you are creating an open-source application under a license compatible with the `GNU AGPL license v3 <https://www.gnu.org/licenses/agpl-3.0.html>`_, you may use ReportBro under the terms of the AGPLv3.

Read more about ReportBro's license options at https://www.reportbro.com/license.