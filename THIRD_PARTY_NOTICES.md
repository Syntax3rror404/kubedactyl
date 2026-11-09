# Third-party notices

Kubedactyl contains or bundles the following third-party material: copied code and assets with their
notices, then every dependency in the tables under "Dependencies". The Helm SDK (self-upgrades,
Apache-2.0) pulls in `github.com/cyphar/filepath-securejoin` (BSD-3-Clause AND MPL-2.0); only its BSD-3-Clause
files are compiled into the panel. NOTICE files of Go modules are at the end.

The build writes this file and the license texts of every listed dependency (from the package itself; the
standard text of the declared license for npm packages without license file) to `third-party-licenses.md`
(`make licenses`): the panel shows it on the page `/licenses` (link "Licenses" in the footer) and the image
contains it as `/usr/share/licenses/kubedactyl/THIRD_PARTY_LICENSES.md`.

## Container image: Alpine Linux

The image `ghcr.io/syntax3rror404/kubedactyl` is based on the unmodified `alpine:3.24.4` image. Its packages
(musl, BusyBox, apk-tools, ca-certificates, …) keep their own licenses, among them GPL-2.0 (BusyBox). Their
sources are published by Alpine Linux: https://gitlab.alpinelinux.org/alpine/aports (the `-stable` branch of that release) and
the source archives referenced there.

## Pterodactyl Wings: configuration file parsers

`backend/internal/configfile` is a port of https://github.com/pterodactyl/wings/tree/develop/parser.

```
MIT License

Copyright (c) 2018 - 2021 Dane Everitt <dane@daneeveritt.com> and Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## shadcn/ui: UI components

`frontend/src/components/ui/*` (except the Magic UI files below), `components/theme-provider.tsx` and
`components/mode-toggle.tsx` come from https://ui.shadcn.com.

```
MIT License

Copyright (c) 2023 shadcn

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Magic UI: effects

`frontend/src/components/ui/magic-card.tsx`, `light-rays.tsx` and `particles.tsx` come from https://magicui.design
(https://github.com/magicuidesign/magicui), installed with the shadcn CLI (`@magicui/…`).

```
MIT License

Copyright (c) Magic UI

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## Geist font

Bundled into the frontend build via @fontsource-variable/geist.

```
Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font) Geist-Italic[wght].ttf: Copyright 2024 The Geist Project Authors (https://github.com/vercel/geist-font)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL


-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION & CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
```

## JetBrains Mono font

Bundled into the frontend build via @fontsource-variable/jetbrains-mono (monospace text, console).

```
Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono) JetBrainsMono-Italic[wght].ttf: Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL


-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION & CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
```

## Small Block font: start-up logo

The logo the panel prints when it starts (`backend/banner.go`) is set in the TOIlet font "Small Block"
(`smblock.tlf`, https://github.com/cacalabs/toilet) by Sam Hocevar, licensed under WTFPL-2.0.

```
            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
                    Version 2, December 2004

 Copyright (C) 2004 Sam Hocevar <sam@hocevar.net>

 Everyone is permitted to copy and distribute verbatim or modified
 copies of this license document, and changing it is allowed as long
 as the name is changed.

            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
   TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION

  0. You just DO WHAT THE FUCK YOU WANT TO.
```

## Dependencies

Every Go module compiled into the panel binary and every npm package in the web interface, with the license
it is used under, tracked by hand, like any other code decision: when a dependency is added, check its license
(repository, license file) and add the row. The versions come from the build (they are in
`third-party-licenses.md`), so updates need no change. The build compares the tables with what it actually
compiles and bundles and fails with the rows to add or remove, and when an npm package declares another license
(`backend/tools/licenses` for Go, `frontend/scripts/licenses-plugin.ts` for npm).

The npm table also lists the build tools whose code the build writes into the web interface: Vite (module
preload helpers) and Rolldown (runtime helpers of the bundles), together with the Vite plugins for React and
Tailwind CSS, which produce the shipped JavaScript and CSS.

### Go modules

| Module | License |
| --- | --- |
| dario.cat/mergo | BSD-3-Clause |
| github.com/BurntSushi/toml | MIT |
| github.com/Jeffail/gabs/v2 | MIT |
| github.com/KyleBanks/depth | MIT |
| github.com/MakeNowJust/heredoc | MIT |
| github.com/Masterminds/goutils | Apache-2.0 |
| github.com/Masterminds/semver/v3 | MIT |
| github.com/Masterminds/sprig/v3 | MIT |
| github.com/Masterminds/squirrel | MIT |
| github.com/ProtonMail/go-crypto | BSD-3-Clause |
| github.com/asaskevich/govalidator | MIT |
| github.com/beevik/etree | BSD-2-Clause |
| github.com/beorn7/perks | MIT |
| github.com/blang/semver/v4 | MIT |
| github.com/cespare/xxhash/v2 | MIT |
| github.com/chai2010/gettext-go | BSD-3-Clause |
| github.com/cloudflare/circl | BSD-3-Clause |
| github.com/coder/websocket | ISC |
| github.com/coreos/go-oidc/v3 | Apache-2.0 |
| github.com/cyphar/filepath-securejoin | BSD-3-Clause AND MPL-2.0 |
| github.com/davecgh/go-spew | ISC |
| github.com/dylibso/observe-sdk/go | Apache-2.0 |
| github.com/emicklei/go-restful/v3 | MIT |
| github.com/evanphx/json-patch/v5 | BSD-3-Clause |
| github.com/exponent-io/jsonpath | MIT |
| github.com/extism/go-sdk | BSD-3-Clause |
| github.com/fatih/color | MIT |
| github.com/fluxcd/cli-utils | Apache-2.0 |
| github.com/fsnotify/fsnotify | BSD-3-Clause |
| github.com/fxamacker/cbor/v2 | MIT |
| github.com/gabriel-vasile/mimetype | MIT |
| github.com/gin-contrib/sse | MIT |
| github.com/gin-gonic/gin | MIT |
| github.com/go-errors/errors | MIT |
| github.com/go-gorp/gorp/v3 | MIT |
| github.com/go-jose/go-jose/v4 | Apache-2.0 |
| github.com/go-logr/logr | Apache-2.0 |
| github.com/go-openapi/jsonpointer | Apache-2.0 |
| github.com/go-openapi/jsonreference | Apache-2.0 |
| github.com/go-openapi/spec | Apache-2.0 |
| github.com/go-openapi/swag | Apache-2.0 |
| github.com/go-openapi/swag/cmdutils | Apache-2.0 |
| github.com/go-openapi/swag/conv | Apache-2.0 |
| github.com/go-openapi/swag/fileutils | Apache-2.0 |
| github.com/go-openapi/swag/jsonutils | Apache-2.0 |
| github.com/go-openapi/swag/loading | Apache-2.0 |
| github.com/go-openapi/swag/mangling | Apache-2.0 |
| github.com/go-openapi/swag/netutils | Apache-2.0 |
| github.com/go-openapi/swag/pools | Apache-2.0 |
| github.com/go-openapi/swag/stringutils | Apache-2.0 |
| github.com/go-openapi/swag/typeutils | Apache-2.0 |
| github.com/go-openapi/swag/yamlutils | Apache-2.0 |
| github.com/go-playground/locales | MIT |
| github.com/go-playground/universal-translator | MIT |
| github.com/go-playground/validator/v10 | MIT |
| github.com/gobwas/glob | MIT |
| github.com/goccy/go-yaml | MIT |
| github.com/google/btree | Apache-2.0 |
| github.com/google/gnostic-models | Apache-2.0 |
| github.com/google/uuid | BSD-3-Clause |
| github.com/gorilla/websocket | BSD-2-Clause |
| github.com/gosuri/uitable | MIT |
| github.com/huandu/xstrings | MIT |
| github.com/iancoleman/strcase | MIT |
| github.com/ianlancetaylor/demangle | BSD-3-Clause |
| github.com/icza/dyno | Apache-2.0 |
| github.com/jmoiron/sqlx | MIT |
| github.com/json-iterator/go | MIT |
| github.com/lann/builder | MIT |
| github.com/lann/ps | MIT |
| github.com/leodido/go-urn | MIT |
| github.com/lib/pq | MIT |
| github.com/liggitt/tabwriter | BSD-3-Clause |
| github.com/magiconair/properties | BSD-2-Clause |
| github.com/mattn/go-colorable | MIT |
| github.com/mattn/go-isatty | MIT |
| github.com/mattn/go-runewidth | MIT |
| github.com/mitchellh/copystructure | MIT |
| github.com/mitchellh/go-wordwrap | MIT |
| github.com/mitchellh/reflectwalk | MIT |
| github.com/moby/spdystream | Apache-2.0 |
| github.com/moby/term | Apache-2.0 |
| github.com/modern-go/concurrent | Apache-2.0 |
| github.com/modern-go/reflect2 | Apache-2.0 |
| github.com/monochromegane/go-gitignore | MIT |
| github.com/munnerz/goautoneg | BSD-3-Clause |
| github.com/opencontainers/go-digest | Apache-2.0 |
| github.com/opencontainers/image-spec | Apache-2.0 |
| github.com/pelletier/go-toml/v2 | MIT |
| github.com/peterbourgon/diskv | MIT |
| github.com/pmezard/go-difflib | BSD-3-Clause |
| github.com/prometheus/client_golang | Apache-2.0 |
| github.com/prometheus/client_model | Apache-2.0 |
| github.com/prometheus/common | Apache-2.0 |
| github.com/prometheus/procfs | Apache-2.0 |
| github.com/quic-go/qpack | MIT |
| github.com/quic-go/quic-go | MIT |
| github.com/robfig/cron/v3 | MIT |
| github.com/rubenv/sql-migrate | MIT |
| github.com/russross/blackfriday/v2 | BSD-2-Clause |
| github.com/santhosh-tekuri/jsonschema/v6 | Apache-2.0 |
| github.com/shopspring/decimal | MIT |
| github.com/spf13/cast | MIT |
| github.com/spf13/cobra | Apache-2.0 |
| github.com/spf13/pflag | BSD-3-Clause |
| github.com/swaggo/files | MIT |
| github.com/swaggo/gin-swagger | MIT |
| github.com/swaggo/swag | MIT |
| github.com/tetratelabs/wabin | Apache-2.0 |
| github.com/tetratelabs/wazero | Apache-2.0 |
| github.com/ugorji/go/codec | MIT |
| github.com/x448/float16 | MIT |
| github.com/xlab/treeprint | MIT |
| go.mongodb.org/mongo-driver/v2 | Apache-2.0 |
| go.opentelemetry.io/proto/otlp | Apache-2.0 |
| go.yaml.in/yaml/v2 | Apache-2.0 AND MIT |
| go.yaml.in/yaml/v3 | Apache-2.0 AND MIT |
| golang.org/x/crypto | BSD-3-Clause |
| golang.org/x/mod | BSD-3-Clause |
| golang.org/x/net | BSD-3-Clause |
| golang.org/x/oauth2 | BSD-3-Clause |
| golang.org/x/sync | BSD-3-Clause |
| golang.org/x/sys | BSD-3-Clause |
| golang.org/x/term | BSD-3-Clause |
| golang.org/x/text | BSD-3-Clause |
| golang.org/x/time | BSD-3-Clause |
| golang.org/x/tools | BSD-3-Clause |
| gomodules.xyz/jsonpatch/v2 | Apache-2.0 |
| google.golang.org/protobuf | BSD-3-Clause |
| gopkg.in/evanphx/json-patch.v4 | BSD-3-Clause |
| gopkg.in/inf.v0 | BSD-3-Clause |
| gopkg.in/ini.v1 | Apache-2.0 |
| helm.sh/helm/v4 | Apache-2.0 |
| k8s.io/api | Apache-2.0 |
| k8s.io/apiextensions-apiserver | Apache-2.0 |
| k8s.io/apimachinery | Apache-2.0 |
| k8s.io/apiserver | Apache-2.0 |
| k8s.io/cli-runtime | Apache-2.0 |
| k8s.io/client-go | Apache-2.0 |
| k8s.io/component-base | Apache-2.0 |
| k8s.io/klog/v2 | Apache-2.0 |
| k8s.io/kube-openapi | Apache-2.0 |
| k8s.io/kubectl | Apache-2.0 |
| k8s.io/streaming | Apache-2.0 |
| k8s.io/utils | Apache-2.0 |
| oras.land/oras-go/v2 | Apache-2.0 |
| sigs.k8s.io/controller-runtime | Apache-2.0 |
| sigs.k8s.io/json | Apache-2.0 AND BSD-3-Clause |
| sigs.k8s.io/kustomize/api | Apache-2.0 |
| sigs.k8s.io/kustomize/kyaml | Apache-2.0 |
| sigs.k8s.io/randfill | Apache-2.0 |
| sigs.k8s.io/structured-merge-diff/v6 | Apache-2.0 |
| sigs.k8s.io/yaml | Apache-2.0 AND MIT AND BSD-3-Clause |

### npm packages

| Package | License |
| --- | --- |
| @babel/runtime | MIT |
| @base-ui/react | MIT |
| @base-ui/utils | MIT |
| @codemirror/autocomplete | MIT |
| @codemirror/commands | MIT |
| @codemirror/lang-angular | MIT |
| @codemirror/lang-cpp | MIT |
| @codemirror/lang-css | MIT |
| @codemirror/lang-go | MIT |
| @codemirror/lang-html | MIT |
| @codemirror/lang-java | MIT |
| @codemirror/lang-javascript | MIT |
| @codemirror/lang-jinja | MIT |
| @codemirror/lang-json | MIT |
| @codemirror/lang-less | MIT |
| @codemirror/lang-liquid | MIT |
| @codemirror/lang-markdown | MIT |
| @codemirror/lang-php | MIT |
| @codemirror/lang-python | MIT |
| @codemirror/lang-rust | MIT |
| @codemirror/lang-sass | MIT |
| @codemirror/lang-sql | MIT |
| @codemirror/lang-vue | MIT |
| @codemirror/lang-wast | MIT |
| @codemirror/lang-xml | MIT |
| @codemirror/lang-yaml | MIT |
| @codemirror/language | MIT |
| @codemirror/language-data | MIT |
| @codemirror/legacy-modes | MIT |
| @codemirror/lint | MIT |
| @codemirror/search | MIT |
| @codemirror/state | MIT |
| @codemirror/streamparser | MIT |
| @codemirror/theme-one-dark | MIT |
| @codemirror/view | MIT |
| @floating-ui/core | MIT |
| @floating-ui/dom | MIT |
| @floating-ui/react-dom | MIT |
| @floating-ui/utils | MIT |
| @fontsource-variable/geist | OFL-1.1 |
| @fontsource-variable/jetbrains-mono | OFL-1.1 |
| @lezer/common | MIT |
| @lezer/cpp | MIT |
| @lezer/css | MIT |
| @lezer/go | MIT |
| @lezer/highlight | MIT |
| @lezer/html | MIT |
| @lezer/java | MIT |
| @lezer/javascript | MIT |
| @lezer/json | MIT |
| @lezer/lr | MIT |
| @lezer/markdown | MIT |
| @lezer/php | MIT |
| @lezer/python | MIT |
| @lezer/rust | MIT |
| @lezer/sass | MIT |
| @lezer/xml | MIT |
| @lezer/yaml | MIT |
| @marijn/find-cluster-break | MIT |
| @reduxjs/toolkit | MIT |
| @remix-run/route-pattern | MIT |
| @tailwindcss/vite | MIT |
| @tanstack/query-core | MIT |
| @tanstack/react-query | MIT |
| @uiw/codemirror-extensions-basic-setup | MIT |
| @uiw/react-codemirror | MIT |
| @ungap/structured-clone | ISC |
| @xterm/addon-fit | MIT |
| @xterm/xterm | MIT |
| attr-accept | MIT |
| bail | MIT |
| ccount | MIT |
| class-variance-authority | Apache-2.0 |
| clsx | MIT |
| cn | MIT |
| comma-separated-tokens | MIT |
| cookie-es | MIT |
| crelt | MIT |
| cronstrue | MIT |
| d3-array | ISC |
| d3-color | ISC |
| d3-format | ISC |
| d3-interpolate | ISC |
| d3-path | ISC |
| d3-scale | ISC |
| d3-shape | ISC |
| d3-time | ISC |
| d3-time-format | ISC |
| decimal.js-light | MIT |
| decode-named-character-reference | MIT |
| devlop | MIT |
| es-toolkit | MIT |
| escape-string-regexp | MIT |
| estree-util-is-identifier-name | MIT |
| eventemitter3 | MIT |
| extend | MIT |
| file-selector | MIT |
| framer-motion | MIT |
| hast-util-to-jsx-runtime | MIT |
| hast-util-whitespace | MIT |
| html-url-attributes | MIT |
| immer | MIT |
| inline-style-parser | MIT |
| internmap | ISC |
| is-plain-obj | MIT |
| longest-streak | MIT |
| lucide-react | ISC |
| markdown-table | MIT |
| mdast-util-find-and-replace | MIT |
| mdast-util-from-markdown | MIT |
| mdast-util-gfm | MIT |
| mdast-util-gfm-autolink-literal | MIT |
| mdast-util-gfm-footnote | MIT |
| mdast-util-gfm-strikethrough | MIT |
| mdast-util-gfm-table | MIT |
| mdast-util-gfm-task-list-item | MIT |
| mdast-util-phrasing | MIT |
| mdast-util-to-hast | MIT |
| mdast-util-to-markdown | MIT |
| mdast-util-to-string | MIT |
| micromark | MIT |
| micromark-core-commonmark | MIT |
| micromark-extension-gfm | MIT |
| micromark-extension-gfm-autolink-literal | MIT |
| micromark-extension-gfm-footnote | MIT |
| micromark-extension-gfm-strikethrough | MIT |
| micromark-extension-gfm-table | MIT |
| micromark-extension-gfm-tagfilter | MIT |
| micromark-extension-gfm-task-list-item | MIT |
| micromark-factory-destination | MIT |
| micromark-factory-label | MIT |
| micromark-factory-space | MIT |
| micromark-factory-title | MIT |
| micromark-factory-whitespace | MIT |
| micromark-util-character | MIT |
| micromark-util-chunked | MIT |
| micromark-util-classify-character | MIT |
| micromark-util-combine-extensions | MIT |
| micromark-util-decode-numeric-character-reference | MIT |
| micromark-util-decode-string | MIT |
| micromark-util-edit-map | MIT |
| micromark-util-encode | MIT |
| micromark-util-html-tag-name | MIT |
| micromark-util-normalize-identifier | MIT |
| micromark-util-resolve-all | MIT |
| micromark-util-sanitize-uri | MIT |
| micromark-util-subtokenize | MIT |
| motion | MIT |
| motion-dom | MIT |
| motion-utils | MIT |
| next-themes | MIT |
| property-information | MIT |
| react | MIT |
| react-dom | MIT |
| react-dropzone | MIT |
| react-is | MIT |
| react-markdown | MIT |
| react-redux | MIT |
| react-router | MIT |
| recharts | MIT |
| redux | MIT |
| redux-thunk | MIT |
| remark-gfm | MIT |
| remark-parse | MIT |
| remark-rehype | MIT |
| reselect | MIT |
| rolldown | MIT |
| scheduler | MIT |
| shadcn | MIT |
| sonner | MIT |
| space-separated-tokens | MIT |
| style-mod | MIT |
| style-to-object | MIT |
| tailwindcss | MIT |
| tiny-invariant | MIT |
| trim-lines | MIT |
| trough | MIT |
| tslib | 0BSD |
| tw-animate-css | MIT |
| unified | MIT |
| unist-util-is | MIT |
| unist-util-position | MIT |
| unist-util-stringify-position | MIT |
| unist-util-visit | MIT |
| unist-util-visit-parents | MIT |
| uqr | MIT |
| use-sync-external-store | MIT |
| vfile | MIT |
| vfile-message | MIT |
| victory-vendor | MIT AND ISC AND BSD-3-Clause |
| vite | MIT |
| w3c-keyname | MIT |
| zwitch | MIT |

## NOTICE files of Go modules

The panel binary is built from Go modules; those with a NOTICE file (Apache License 2.0, section 4(d))
are reproduced here. The list follows `go version -m bin/kubedactyl`.

### github.com/coreos/go-oidc/v3 v3.21.0

```
CoreOS Project
Copyright 2014 CoreOS, Inc

This product includes software developed at CoreOS, Inc.
(http://www.coreos.com/).
```

### github.com/go-openapi/jsonpointer v1.0.0

```
Copyright 2015-2025 go-swagger maintainers

// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

This software library, github.com/go-openapi/jsonpointer, includes software developed
by the go-swagger and go-openapi maintainers ("go-swagger maintainers").

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this software except in compliance with the License.

You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0.

This software is copied from, derived from, and inspired by other original software products.
It ships with copies of other software which license terms are recalled below.

The original software was authored on 25-02-2013 by sigu-399 (https://github.com/sigu-399, sigu.399@gmail.com).

github.com/sigu-399/jsonpointer
===========================

// SPDX-FileCopyrightText: Copyright 2013 sigu-399 ( https://github.com/sigu-399 )
// SPDX-License-Identifier: Apache-2.0

Copyright 2013 sigu-399 ( https://github.com/sigu-399 )

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### github.com/go-openapi/jsonreference v1.0.0

```
Copyright 2015-2025 go-swagger maintainers

// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

This software library, github.com/go-openapi/jsonreference, includes software developed
by the go-swagger and go-openapi maintainers ("go-swagger maintainers").

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this software except in compliance with the License.

You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0.

This software is copied from, derived from, and inspired by other original software products.
It ships with copies of other software which license terms are recalled below.

The original software was authored on 25-02-2013 by sigu-399 (https://github.com/sigu-399, sigu.399@gmail.com).

github.com/sigh-399/jsonreference
===========================

// SPDX-FileCopyrightText: Copyright 2013 sigu-399 ( https://github.com/sigu-399 )
// SPDX-License-Identifier: Apache-2.0

Copyright 2013 sigu-399 ( https://github.com/sigu-399 )

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### github.com/moby/spdystream v0.5.1

```
SpdyStream
Copyright 2014-2021 Docker Inc.

This product includes software developed at
Docker Inc. (https://www.docker.com/).

SPDY implementation (spdy/)

The spdy directory contains code derived from the Go project (golang.org/x/net).

Copyright 2009-2013 The Go Authors.
Licensed under the BSD 3-Clause License.

Modifications Copyright 2014-2021 Docker Inc.

The BSD license text and Go patent grant are included in
spdy/LICENSE and spdy/PATENTS.
```

### github.com/prometheus/client_golang v1.24.0

```
Prometheus instrumentation library for Go applications
Copyright 2012-2015 The Prometheus Authors

This product includes software developed at
SoundCloud Ltd. (http://soundcloud.com/).


The following components are included in this product:

perks - a fork of https://github.com/bmizerany/perks
https://github.com/beorn7/perks
Copyright 2013-2015 Blake Mizerany, Björn Rabenstein
See https://github.com/beorn7/perks/blob/master/README.md for license details.

Go support for Protocol Buffers - Google's data interchange format
http://github.com/golang/protobuf/
Copyright 2010 The Go Authors
See source code for license details.
```

### github.com/prometheus/client_model v0.6.2

```
Data model artifacts for Prometheus.
Copyright 2012-2015 The Prometheus Authors

This product includes software developed at
SoundCloud Ltd. (http://soundcloud.com/).
```

### github.com/prometheus/common v0.70.0

```
Common libraries shared by Prometheus Go components.
Copyright 2015 The Prometheus Authors

This product includes software developed at
SoundCloud Ltd. (http://soundcloud.com/).
```

### github.com/tetratelabs/wazero v1.12.0

```
wazero
Copyright 2020-2023 wazero authors
```

### go.yaml.in/yaml/v2 v2.4.4

```
Copyright 2011-2016 Canonical Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### go.yaml.in/yaml/v3 v3.0.5

```
Copyright 2011-2016 Canonical Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### sigs.k8s.io/randfill v1.0.0

```
When donating the randfill project to the CNCF, we could not reach all the
gofuzz contributors to sign the CNCF CLA. As such, according to the CNCF rules
to donate a repository, we must add a NOTICE referencing section 7 of the CLA
with a list of developers who could not be reached.

`7. Should You wish to submit work that is not Your original creation, You may
submit it to the Foundation separately from any Contribution, identifying the
complete details of its source and of any license or other restriction
(including, but not limited to, related patents, trademarks, and license
agreements) of which you are personally aware, and conspicuously marking the
work as "Submitted on behalf of a third-party: [named here]".`

Submitted on behalf of a third-party: @dnephin (Daniel Nephin)
Submitted on behalf of a third-party: @AlekSi (Alexey Palazhchenko)
Submitted on behalf of a third-party: @bbigras (Bruno Bigras)
Submitted on behalf of a third-party: @samirkut (Samir)
Submitted on behalf of a third-party: @posener (Eyal Posener)
Submitted on behalf of a third-party: @Ashikpaul (Ashik Paul)
Submitted on behalf of a third-party: @kwongtailau (Kwongtai)
Submitted on behalf of a third-party: @ericcornelissen (Eric Cornelissen)
Submitted on behalf of a third-party: @eclipseo (Robert-André Mauchin)
Submitted on behalf of a third-party: @yanzhoupan (Andrew Pan)
Submitted on behalf of a third-party: @STRRL (Zhiqiang ZHOU)
Submitted on behalf of a third-party: @disconnect3d (Disconnect3d)
```
