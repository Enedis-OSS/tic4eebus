<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->
# Contributing

[🇫🇷 Français](CONTRIBUTING.md) | [🇺🇸 English](CONTRIBUTING.en.md)

## Summary

* [How to contribute to the documentation](#doc)
* [How to make a Pull Request](#pr)
* [Code convention](#code)
* [Test convention](#test)
* [Branch convention](#branch)
* [Commit changes](#commit)
* [Dependency management](#dep)
* [Build Process](#build)
* [Release Management](#release)
* [Releasing](#releasing)
* [Licensing](#oss)


## <a name="doc"></a> How to contribute to the documentation

To contribute to this documentation (README, CONTRIBUTING, etc.), we conform to the [CommonMark Spec](http://spec.commonmark.org/0.27/)

* [https://www.makeareadme.com/#suggestions-for-a-good-readme](https://www.makeareadme.com/#suggestions-for-a-good-readme)
* [https://help.github.com/en/articles/setting-guidelines-for-repository-contributors](https://help.github.com/en/articles/setting-guidelines-for-repository-contributors)


## <a name="pr"></a> How to make a Pull Request

1. Fork the repository and keep active sync on our repo
2. Create your working branches conforming to [conventional branch](https://conventional-branch.github.io/).
    * **WARNING** - Do not modify the main branch nor any of our branches since it will break the automatic sync
3. When you are done, fetch all and rebase your branch onto our main or any other of ours
    * ex. on your branch, do:
        * `git fetch --all --prune`
        * `git rebase --no-ff origin/main`
4. Test your changes and make sure everything is working
5. Submit your Pull Request to the dev branch (main branch is only allowed for project owners)
    * Do not forget to add reviewers! Check out the last authors of the code you modified and add them.
    * In case of doubts, here are active contributors :
        * Jehan BOUSCH


## <a name="code"></a> Code convention

### Best Practices

As a general rule you have to follow the [go style guide](https://google.github.io/styleguide/go/).
Read the [best pratices](https://google.github.io/styleguide/go/best-practices) section before making a Pull Request.

### Acronyms

Whenever an acronym is included as part of a type name or method name, keep the first
letter of the acronym uppercase and use lowercase for the rest of the acronym. Otherwise,
it becomes _impossible_ to perform camel-cased searches in IDEs, and it becomes
potentially very difficult for mere humans to read or reason about the element without
reading documentation (if documentation even exists).

Consider for example a use case needing to support an HTTP URL. Calling the method
`GetHTTPURL()` is absolutely horrible in terms of usability; whereas, `GetHttpUrl()` is
great in terms of usability. The same applies for types `HTTPURLProvider` vs
`HttpUrlProvider`, etc.

Whenever an acronym is included as part of a field name or parameter name:

* If the acronym comes at the start of the field or parameter name, use lowercase for the
  entire acronym
    * for example, `var url string`.

* Otherwise, keep the first letter of the acronym uppercase and use lowercase for the
  rest of the acronym
    * for example, `var defaultUrl string`.

### Formatting

All Go source files must conform to the format outputted by the gofmt tool.

### Godoc

All Go source files must use the [godoc formatting](https://google.github.io/styleguide/go/best-practices#godoc-formatting).

## <a name="test"></a> Test convention

To run all the tests of the project use the following command :

   ```
   go test ./...
   ```

### Naming

* Tests are located in the **same directory** as the code under test.
* A test file name should end with **_test.go**.
* All test functions must start with a `Test` prefix and use PascalCase.

### Assertions

* Use assertion package like `github.com/stretchr/testify/assert` wherever possible.

### Mocking

* Use `github.com/stretchr/testify/mock` to mock your interfaces

## <a name="branch"></a> Branch convention

We conform to [conventional branch](https://conventional-branch.github.io/).

## <a name="commit"></a> Commit changes

We conform to [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/).

## <a name="dep"></a> Dependency management

Project dependencies are listed in the go.mod file. The go.sum file, on the other hand, contains the cryptographic checksums of the content of specific module versions, including both direct and indirect dependencies.

To see modules really "used" by the application, use the following command :
   ```
   go list -m all
   ```

To update dependencies to their latest versions use the command :
   ```
   go get -u ./...
   ```

To clean up and synchronize the go.mod and go.sum files with the actual code in the project use :
   ```
   go mod tidy
   ```

For more detailed documentation please refer to [official managing-dependencies documentation](https://go.dev/doc/modules/managing-dependencies).

## <a name="build"></a> Build Process

To build the application tic4eebus in the project directory use the following command :

   ```
   go build -o tic4eebus cmd/tic4eebus/main.go
   ```

## <a name="release"></a> Release Management

Release management is exclusively done on GitHub

## <a name="releasing"></a> Releasing

tic4eebus releases are available only on GitHub.

### Update Changelog file

Always update the changelog before creating a release. This ensures the changes are documented for the release being created.
Refer to this page for guidance on automatically generated release notes:  [Automatically generated release notes](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes)

Do not hesitate to update the release note generated especially the titles of pull request :)
Use it to update [CHANGELOG.md](https://github.com/Enedis-OSS/tic4eebus/blob/main/CHANGELOG.md)


### Releasing version only on GitHub

#### General information

- Releases to GitHub only update the last digit of the version (e.g., `2.7.1.1` or `2.9.4.2`).
- The subsequent snapshot version remains unchanged.
- The `CHANGELOG.md` file is committed to the tag.
- Pull requests are not required for releases to GitHub.

#### Step by step

```shell
git switch -c release/<RELEASE_VERSION>
git add CHANGELOG.md (see Update Changelog file section)
Update manually `VERSION` in [cmd/tic4eebus/main.go](https://github.com/Enedis-OSS/tic4eebus/blob/main/cmd/tic4eebus/main.go)
git add .
git diff --staged
git commit -m "chore: Release <RELEASE_VERSION>"
```

- Then tag and push.
```shell
git tag <TAG_VERSION>
git push origin <TAG_VERSION>
```

- Update the release note on [github](https://github.com/Enedis-OSS/tic4eebus/releases)


## <a name="oss"></a> Licensing

We choose to apply the Apache License 2.0 (ALv2) : [http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

As for any project, license compatibility issues may arise and should be taken care of.

Concrete instructions and tooling to keep tic4eebus ALv2 compliant and limit licensing issues are to be found below.

However, we acknowledge topic's complexity, mistakes might be done and we might not get it 100% right.

Still, we strive to be compliant and be fair, meaning, we do our best in good faith.

As such, we welcome any advice and change request.


To any contributor, we strongly recommend further reading and personal research :
* [http://www.apache.org/licenses/](http://www.apache.org/licenses/)
* [http://www.apache.org/legal/](http://www.apache.org/legal/)
* [http://apache.org/legal/resolved.html](http://apache.org/legal/resolved.html)
* [http://www.apache.org/dev/apply-license.html](http://www.apache.org/dev/apply-license.html)
* [http://www.apache.org/legal/src-headers.html](http://apache.org/legal/src-headers.html)
* [http://www.apache.org/legal/release-policy.html](http://www.apache.org/legal/release-policy.html)
* [http://www.apache.org/dev/licensing-howto.html](http://www.apache.org/dev/licensing-howto.html)

* [Why is LGPL not allowed](https://issues.apache.org/jira/browse/LEGAL-192)
* https://issues.apache.org/jira/projects/LEGAL/issues/

* General news : [https://opensource.com/tags/law](https://opensource.com/tags/law)

### How to manage license compatibility

When adding a new dependency, **one should check its license and all its transitive dependencies** licenses.

ALv2 license compatibility as defined by the ASF can be found here : [http://apache.org/legal/resolved.html](http://apache.org/legal/resolved.html)

3 categories are defined :
* [Category A](https://www.apache.org/legal/resolved.html#category-a) : Contains all compatibles licenses.
* [Category B](https://www.apache.org/legal/resolved.html#category-b) : Contains compatibles licenses under certain conditions.
* [Category X](https://www.apache.org/legal/resolved.html#category-x) : Contains all incompatibles licenses which must be avoid at all cost.

__As far as we understand :__

If, by any mean, your contribution should rely on a Category X dependency, then you must provide a way to modularize it
and make it's use optional to tic4eebus, as a plugin.

You may distribute your plugin under the terms of the Category X license.

Any distribution of tic4eebus bundled with your plugin will probably be done under the terms of the Category X license.

But _"you can provide the user with instructions on how to obtain and install the non-included"_ plugin.

__References :__
- [Optional](https://www.apache.org/legal/resolved.html#optional)
- [Prohibited](https://www.apache.org/legal/resolved.html#prohibited)

### How to comply with Redistribution and Attribution clauses

Lots of licenses place conditions on redistribution and attribution, including ALv2.

__References :__
* http://mail-archives.apache.org/mod_mbox/www-legal-discuss/201502.mbox/%3CCAAS6%3D7gzsAYZMT5mar_nfy9egXB1t3HendDQRMUpkA6dqvhr7w%40mail.gmail.com%3E
* http://mail-archives.apache.org/mod_mbox/www-legal-discuss/201501.mbox/%3CCAAS6%3D7jJoJMkzMRpSdJ6kAVSZCvSfC5aRD0eMyGzP_rzWyE73Q%40mail.gmail.com%3E

#### LICENSE file
##### In Source distribution

This file contains :
* the complete ALv2 license.
* list dependencies and points to their respective license file
    * Example :
      _This product bundles SuperWidget 1.2.3, which is available under a
      "3-clause BSD" license.  For deails, see deps/superwidget/_
* do not list dependencies under the ALv2

#### NOTICE file

##### In source distribution

_The NOTICE file is not for conveying information to downstream consumers -- it
is a way to *compel* downstream consumers to *relay* certain required notices._

### Unresolved questions - HELP WANTED -
* Should test dependencies be taken into account for source distribution ?
    * It appears to be YES
* Should build time dependencies be taken into account ?
    * It appears to be NO but might depend on the actual stuff done by this dependency
