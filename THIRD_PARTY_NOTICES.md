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

The image `ghcr.io/syntax3rror404/kubedactyl` is based on the unmodified `alpine:3.24.2` image. Its packages
(musl, BusyBox, apk-tools, ca-certificates, …) keep their own licenses, among them GPL-2.0 (BusyBox). Their
sources are published by Alpine Linux: https://gitlab.alpinelinux.org/alpine/aports (branch `3.24-stable`) and
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
it is used under, tracked by hand, like any other code decision: when a dependency is added or updated,
check its license (repository, license file) and change the row. The build compares the tables with what it
actually compiles and bundles and fails with the rows to add, update or remove (`backend/tools/licenses` for
Go, `frontend/scripts/licenses-plugin.ts` for npm).

The npm table also lists the build tools whose code the build writes into the web interface: Vite (module
preload helpers) and Rolldown (runtime helpers of the bundles), together with the Vite plugins for React and
Tailwind CSS, which produce the shipped JavaScript and CSS.

### Go modules

| Module | Version | License |
| --- | --- | --- |
| dario.cat/mergo | v1.0.1 | BSD-3-Clause |
| github.com/BurntSushi/toml | v1.6.0 | MIT |
| github.com/Jeffail/gabs/v2 | v2.7.0 | MIT |
| github.com/KyleBanks/depth | v1.2.1 | MIT |
| github.com/MakeNowJust/heredoc | v1.0.0 | MIT |
| github.com/Masterminds/goutils | v1.1.1 | Apache-2.0 |
| github.com/Masterminds/semver/v3 | v3.5.0 | MIT |
| github.com/Masterminds/sprig/v3 | v3.3.0 | MIT |
| github.com/Masterminds/squirrel | v1.5.4 | MIT |
| github.com/ProtonMail/go-crypto | v1.4.1 | BSD-3-Clause |
| github.com/asaskevich/govalidator | v0.0.0-20230301143203-a9d515a09cc2 | MIT |
| github.com/beevik/etree | v1.8.1 | BSD-2-Clause |
| github.com/beorn7/perks | v1.0.1 | MIT |
| github.com/blang/semver/v4 | v4.0.0 | MIT |
| github.com/cespare/xxhash/v2 | v2.3.0 | MIT |
| github.com/chai2010/gettext-go | v1.0.2 | BSD-3-Clause |
| github.com/cloudflare/circl | v1.6.3 | BSD-3-Clause |
| github.com/coder/websocket | v1.8.15 | ISC |
| github.com/cyphar/filepath-securejoin | v0.7.0 | BSD-3-Clause AND MPL-2.0 |
| github.com/davecgh/go-spew | v1.1.2-0.20180830191138-d8f796af33cc | ISC |
| github.com/dylibso/observe-sdk/go | v0.0.0-20240819160327-2d926c5d788a | Apache-2.0 |
| github.com/emicklei/go-restful/v3 | v3.13.0 | MIT |
| github.com/evanphx/json-patch/v5 | v5.9.11 | BSD-3-Clause |
| github.com/exponent-io/jsonpath | v0.0.0-20210407135951-1de76d718b3f | MIT |
| github.com/extism/go-sdk | v1.7.1 | BSD-3-Clause |
| github.com/fatih/color | v1.19.0 | MIT |
| github.com/fluxcd/cli-utils | v1.2.2 | Apache-2.0 |
| github.com/fsnotify/fsnotify | v1.9.0 | BSD-3-Clause |
| github.com/fxamacker/cbor/v2 | v2.9.1 | MIT |
| github.com/gabriel-vasile/mimetype | v1.4.12 | MIT |
| github.com/gin-contrib/sse | v1.1.0 | MIT |
| github.com/gin-gonic/gin | v1.12.0 | MIT |
| github.com/go-errors/errors | v1.5.1 | MIT |
| github.com/go-gorp/gorp/v3 | v3.1.0 | MIT |
| github.com/go-logr/logr | v1.4.4 | Apache-2.0 |
| github.com/go-openapi/jsonpointer | v1.0.0 | Apache-2.0 |
| github.com/go-openapi/jsonreference | v1.0.0 | Apache-2.0 |
| github.com/go-openapi/spec | v0.20.4 | Apache-2.0 |
| github.com/go-openapi/swag | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/cmdutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/conv | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/fileutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/jsonutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/loading | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/mangling | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/netutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/pools | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/stringutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/typeutils | v0.27.1 | Apache-2.0 |
| github.com/go-openapi/swag/yamlutils | v0.27.1 | Apache-2.0 |
| github.com/go-playground/locales | v0.14.1 | MIT |
| github.com/go-playground/universal-translator | v0.18.1 | MIT |
| github.com/go-playground/validator/v10 | v10.30.1 | MIT |
| github.com/gobwas/glob | v0.2.3 | MIT |
| github.com/goccy/go-yaml | v1.19.2 | MIT |
| github.com/google/btree | v1.1.3 | Apache-2.0 |
| github.com/google/gnostic-models | v0.7.1 | Apache-2.0 |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause |
| github.com/gorilla/websocket | v1.5.4-0.20250319132907-e064f32e3674 | BSD-2-Clause |
| github.com/gosuri/uitable | v0.0.4 | MIT |
| github.com/huandu/xstrings | v1.5.0 | MIT |
| github.com/iancoleman/strcase | v0.3.0 | MIT |
| github.com/ianlancetaylor/demangle | v0.0.0-20240805132620-81f5be970eca | BSD-3-Clause |
| github.com/icza/dyno | v0.0.0-20260912175837-51f8ab52fd15 | Apache-2.0 |
| github.com/jmoiron/sqlx | v1.4.0 | MIT |
| github.com/json-iterator/go | v1.1.12 | MIT |
| github.com/lann/builder | v0.0.0-20180802200727-47ae307949d0 | MIT |
| github.com/lann/ps | v0.0.0-20150810152359-62de8c46ede0 | MIT |
| github.com/leodido/go-urn | v1.4.0 | MIT |
| github.com/lib/pq | v1.12.3 | MIT |
| github.com/liggitt/tabwriter | v0.0.0-20181228230101-89fcab3d43de | BSD-3-Clause |
| github.com/magiconair/properties | v1.18.12 | BSD-2-Clause |
| github.com/mattn/go-colorable | v0.1.14 | MIT |
| github.com/mattn/go-isatty | v0.0.20 | MIT |
| github.com/mattn/go-runewidth | v0.0.9 | MIT |
| github.com/mitchellh/copystructure | v1.2.0 | MIT |
| github.com/mitchellh/go-wordwrap | v1.0.1 | MIT |
| github.com/mitchellh/reflectwalk | v1.0.2 | MIT |
| github.com/moby/spdystream | v0.5.1 | Apache-2.0 |
| github.com/moby/term | v0.5.2 | Apache-2.0 |
| github.com/modern-go/concurrent | v0.0.0-20180306012644-bacd9c7ef1dd | Apache-2.0 |
| github.com/modern-go/reflect2 | v1.0.3-0.20250322232337-35a7c28c31ee | Apache-2.0 |
| github.com/monochromegane/go-gitignore | v0.0.0-20200626010858-205db1a8cc00 | MIT |
| github.com/munnerz/goautoneg | v0.0.0-20191010083416-a7dc8b61c822 | BSD-3-Clause |
| github.com/opencontainers/go-digest | v1.0.0 | Apache-2.0 |
| github.com/opencontainers/image-spec | v1.1.1 | Apache-2.0 |
| github.com/pelletier/go-toml/v2 | v2.2.4 | MIT |
| github.com/peterbourgon/diskv | v2.0.1+incompatible | MIT |
| github.com/pmezard/go-difflib | v1.0.1-0.20181226105442-5d4384ee4fb2 | BSD-3-Clause |
| github.com/prometheus/client_golang | v1.24.0 | Apache-2.0 |
| github.com/prometheus/client_model | v0.6.2 | Apache-2.0 |
| github.com/prometheus/common | v0.70.0 | Apache-2.0 |
| github.com/prometheus/procfs | v0.21.1 | Apache-2.0 |
| github.com/quic-go/qpack | v0.6.0 | MIT |
| github.com/quic-go/quic-go | v0.63.0 | MIT |
| github.com/robfig/cron/v3 | v3.0.1 | MIT |
| github.com/rubenv/sql-migrate | v1.8.1 | MIT |
| github.com/russross/blackfriday/v2 | v2.1.0 | BSD-2-Clause |
| github.com/santhosh-tekuri/jsonschema/v6 | v6.0.3 | Apache-2.0 |
| github.com/shopspring/decimal | v1.4.0 | MIT |
| github.com/spf13/cast | v1.7.0 | MIT |
| github.com/spf13/cobra | v1.10.2 | Apache-2.0 |
| github.com/spf13/pflag | v1.0.10 | BSD-3-Clause |
| github.com/swaggo/files | v1.0.1 | MIT |
| github.com/swaggo/gin-swagger | v1.6.1 | MIT |
| github.com/swaggo/swag | v1.16.6 | MIT |
| github.com/tetratelabs/wabin | v0.0.0-20230304001439-f6f874872834 | Apache-2.0 |
| github.com/tetratelabs/wazero | v1.12.0 | Apache-2.0 |
| github.com/ugorji/go/codec | v1.3.1 | MIT |
| github.com/x448/float16 | v0.8.4 | MIT |
| github.com/xlab/treeprint | v1.2.0 | MIT |
| go.mongodb.org/mongo-driver/v2 | v2.5.0 | Apache-2.0 |
| go.opentelemetry.io/proto/otlp | v1.10.0 | Apache-2.0 |
| go.yaml.in/yaml/v2 | v2.4.4 | Apache-2.0 AND MIT |
| go.yaml.in/yaml/v3 | v3.0.5 | Apache-2.0 AND MIT |
| golang.org/x/crypto | v0.57.0 | BSD-3-Clause |
| golang.org/x/mod | v0.41.0 | BSD-3-Clause |
| golang.org/x/net | v0.59.0 | BSD-3-Clause |
| golang.org/x/oauth2 | v0.36.0 | BSD-3-Clause |
| golang.org/x/sync | v0.23.0 | BSD-3-Clause |
| golang.org/x/sys | v0.48.0 | BSD-3-Clause |
| golang.org/x/term | v0.46.0 | BSD-3-Clause |
| golang.org/x/text | v0.42.0 | BSD-3-Clause |
| golang.org/x/time | v0.16.0 | BSD-3-Clause |
| golang.org/x/tools | v0.51.0 | BSD-3-Clause |
| gomodules.xyz/jsonpatch/v2 | v2.4.0 | Apache-2.0 |
| google.golang.org/protobuf | v1.36.12-0.20260120151049-f2248ac996af | BSD-3-Clause |
| gopkg.in/evanphx/json-patch.v4 | v4.13.0 | BSD-3-Clause |
| gopkg.in/inf.v0 | v0.9.1 | BSD-3-Clause |
| gopkg.in/ini.v1 | v1.67.3 | Apache-2.0 |
| helm.sh/helm/v4 | v4.3.0 | Apache-2.0 |
| k8s.io/api | v0.37.1 | Apache-2.0 |
| k8s.io/apiextensions-apiserver | v0.37.1 | Apache-2.0 |
| k8s.io/apimachinery | v0.37.1 | Apache-2.0 |
| k8s.io/apiserver | v0.37.1 | Apache-2.0 |
| k8s.io/cli-runtime | v0.37.1 | Apache-2.0 |
| k8s.io/client-go | v0.37.1 | Apache-2.0 |
| k8s.io/component-base | v0.37.1 | Apache-2.0 |
| k8s.io/klog/v2 | v2.140.0 | Apache-2.0 |
| k8s.io/kube-openapi | v0.0.0-20260721132016-d427ff9ee9ad | Apache-2.0 |
| k8s.io/kubectl | v0.37.0 | Apache-2.0 |
| k8s.io/streaming | v0.37.1 | Apache-2.0 |
| k8s.io/utils | v0.0.0-20260707023825-cf1189d6abe3 | Apache-2.0 |
| oras.land/oras-go/v2 | v2.6.2 | Apache-2.0 |
| sigs.k8s.io/controller-runtime | v0.25.2 | Apache-2.0 |
| sigs.k8s.io/json | v0.0.0-20250730193827-2d320260d730 | Apache-2.0 AND BSD-3-Clause |
| sigs.k8s.io/kustomize/api | v0.21.1 | Apache-2.0 |
| sigs.k8s.io/kustomize/kyaml | v0.21.1 | Apache-2.0 |
| sigs.k8s.io/randfill | v1.0.0 | Apache-2.0 |
| sigs.k8s.io/structured-merge-diff/v6 | v6.4.2 | Apache-2.0 |
| sigs.k8s.io/yaml | v1.6.0 | Apache-2.0 AND MIT AND BSD-3-Clause |

### npm packages

| Package | Version | License |
| --- | --- | --- |
| @babel/runtime | 7.29.7 | MIT |
| @codemirror/autocomplete | 6.20.3 | MIT |
| @codemirror/commands | 6.11.1 | MIT |
| @codemirror/lang-angular | 0.1.4 | MIT |
| @codemirror/lang-cpp | 6.0.3 | MIT |
| @codemirror/lang-css | 6.3.1 | MIT |
| @codemirror/lang-go | 6.0.1 | MIT |
| @codemirror/lang-html | 6.4.12 | MIT |
| @codemirror/lang-java | 6.0.2 | MIT |
| @codemirror/lang-javascript | 6.2.5 | MIT |
| @codemirror/lang-jinja | 6.0.1 | MIT |
| @codemirror/lang-json | 6.0.2 | MIT |
| @codemirror/lang-less | 6.0.2 | MIT |
| @codemirror/lang-liquid | 6.3.2 | MIT |
| @codemirror/lang-markdown | 6.5.2 | MIT |
| @codemirror/lang-php | 6.0.2 | MIT |
| @codemirror/lang-python | 6.2.1 | MIT |
| @codemirror/lang-rust | 6.0.2 | MIT |
| @codemirror/lang-sass | 6.0.2 | MIT |
| @codemirror/lang-sql | 6.10.0 | MIT |
| @codemirror/lang-vue | 0.1.3 | MIT |
| @codemirror/lang-wast | 6.0.2 | MIT |
| @codemirror/lang-xml | 6.1.0 | MIT |
| @codemirror/lang-yaml | 6.1.3 | MIT |
| @codemirror/language | 6.13.1 | MIT |
| @codemirror/language-data | 6.5.2 | MIT |
| @codemirror/legacy-modes | 6.5.4 | MIT |
| @codemirror/lint | 6.9.7 | MIT |
| @codemirror/search | 6.7.2 | MIT |
| @codemirror/state | 6.7.6 | MIT |
| @codemirror/streamparser | 6.0.0 | MIT |
| @codemirror/theme-one-dark | 6.1.3 | MIT |
| @codemirror/view | 6.43.14 | MIT |
| @floating-ui/core | 1.8.0 | MIT |
| @floating-ui/dom | 1.8.0 | MIT |
| @floating-ui/react-dom | 2.1.9 | MIT |
| @floating-ui/utils | 0.2.12 | MIT |
| @fontsource-variable/geist | 5.3.0 | OFL-1.1 |
| @fontsource-variable/jetbrains-mono | 5.3.0 | OFL-1.1 |
| @lezer/common | 1.5.3 | MIT |
| @lezer/cpp | 1.1.6 | MIT |
| @lezer/css | 1.3.8 | MIT |
| @lezer/go | 1.0.1 | MIT |
| @lezer/highlight | 1.2.4 | MIT |
| @lezer/html | 1.3.13 | MIT |
| @lezer/java | 1.1.4 | MIT |
| @lezer/javascript | 1.5.5 | MIT |
| @lezer/json | 1.0.3 | MIT |
| @lezer/lr | 1.4.10 | MIT |
| @lezer/markdown | 1.7.2 | MIT |
| @lezer/php | 1.0.6 | MIT |
| @lezer/python | 1.1.19 | MIT |
| @lezer/rust | 1.0.3 | MIT |
| @lezer/sass | 1.1.0 | MIT |
| @lezer/xml | 1.0.6 | MIT |
| @lezer/yaml | 1.0.4 | MIT |
| @marijn/find-cluster-break | 1.0.4 | MIT |
| @radix-ui/number | 1.1.3 | MIT |
| @radix-ui/primitive | 1.1.7 | MIT |
| @radix-ui/react-accessible-icon | 1.1.16 | MIT |
| @radix-ui/react-accordion | 1.2.21 | MIT |
| @radix-ui/react-alert-dialog | 1.1.24 | MIT |
| @radix-ui/react-arrow | 1.1.16 | MIT |
| @radix-ui/react-aspect-ratio | 1.1.16 | MIT |
| @radix-ui/react-avatar | 1.2.7 | MIT |
| @radix-ui/react-checkbox | 1.3.12 | MIT |
| @radix-ui/react-collapsible | 1.1.21 | MIT |
| @radix-ui/react-collection | 1.1.16 | MIT |
| @radix-ui/react-compose-refs | 1.1.5 | MIT |
| @radix-ui/react-context | 1.2.2 | MIT |
| @radix-ui/react-context-menu | 2.3.8 | MIT |
| @radix-ui/react-dialog | 1.2.0 | MIT |
| @radix-ui/react-direction | 1.1.5 | MIT |
| @radix-ui/react-dismissable-layer | 1.1.20 | MIT |
| @radix-ui/react-dropdown-menu | 2.1.25 | MIT |
| @radix-ui/react-focus-guards | 1.1.6 | MIT |
| @radix-ui/react-focus-scope | 1.2.0 | MIT |
| @radix-ui/react-form | 0.2.0 | MIT |
| @radix-ui/react-hover-card | 1.1.24 | MIT |
| @radix-ui/react-id | 1.1.4 | MIT |
| @radix-ui/react-label | 2.1.16 | MIT |
| @radix-ui/react-menu | 2.1.25 | MIT |
| @radix-ui/react-menubar | 1.1.25 | MIT |
| @radix-ui/react-navigation-menu | 1.3.0 | MIT |
| @radix-ui/react-one-time-password-field | 0.1.17 | MIT |
| @radix-ui/react-password-toggle-field | 0.1.12 | MIT |
| @radix-ui/react-popover | 1.2.0 | MIT |
| @radix-ui/react-popper | 1.3.8 | MIT |
| @radix-ui/react-portal | 1.1.18 | MIT |
| @radix-ui/react-presence | 1.1.11 | MIT |
| @radix-ui/react-primitive | 2.1.11 | MIT |
| @radix-ui/react-progress | 1.1.17 | MIT |
| @radix-ui/react-radio-group | 1.4.8 | MIT |
| @radix-ui/react-roving-focus | 1.1.20 | MIT |
| @radix-ui/react-scroll-area | 1.3.0 | MIT |
| @radix-ui/react-select | 2.3.8 | MIT |
| @radix-ui/react-separator | 1.1.16 | MIT |
| @radix-ui/react-slider | 1.5.0 | MIT |
| @radix-ui/react-slot | 1.4.0 | MIT |
| @radix-ui/react-switch | 1.3.8 | MIT |
| @radix-ui/react-tabs | 1.1.22 | MIT |
| @radix-ui/react-toast | 1.2.24 | MIT |
| @radix-ui/react-toggle | 1.1.19 | MIT |
| @radix-ui/react-toggle-group | 1.1.20 | MIT |
| @radix-ui/react-toolbar | 1.1.20 | MIT |
| @radix-ui/react-tooltip | 1.3.0 | MIT |
| @radix-ui/react-use-callback-ref | 1.1.4 | MIT |
| @radix-ui/react-use-controllable-state | 1.2.6 | MIT |
| @radix-ui/react-use-effect-event | 0.0.5 | MIT |
| @radix-ui/react-use-is-hydrated | 0.1.3 | MIT |
| @radix-ui/react-use-layout-effect | 1.1.4 | MIT |
| @radix-ui/react-use-previous | 1.1.4 | MIT |
| @radix-ui/react-use-size | 1.1.5 | MIT |
| @radix-ui/react-visually-hidden | 1.2.12 | MIT |
| @reduxjs/toolkit | 2.12.0 | MIT |
| @remix-run/route-pattern | 0.22.1 | MIT |
| @tailwindcss/vite | 4.3.3 | MIT |
| @tanstack/query-core | 5.104.1 | MIT |
| @tanstack/react-query | 5.104.1 | MIT |
| @uiw/codemirror-extensions-basic-setup | 4.25.12 | MIT |
| @uiw/react-codemirror | 4.25.12 | MIT |
| @ungap/structured-clone | 1.4.0 | ISC |
| @vitejs/plugin-react | 6.1.1 | MIT |
| @xterm/addon-fit | 0.11.0 | MIT |
| @xterm/xterm | 6.0.0 | MIT |
| aria-hidden | 1.2.6 | MIT |
| attr-accept | 4.0.0 | MIT |
| bail | 2.0.2 | MIT |
| ccount | 2.0.1 | MIT |
| class-variance-authority | 0.7.1 | Apache-2.0 |
| clsx | 2.1.1 | MIT |
| cn | 0.4.0 | MIT |
| comma-separated-tokens | 2.0.3 | MIT |
| cookie-es | 3.1.1 | MIT |
| crelt | 1.0.7 | MIT |
| cronstrue | 3.30.0 | MIT |
| d3-array | 3.2.4 | ISC |
| d3-color | 3.1.0 | ISC |
| d3-format | 3.1.2 | ISC |
| d3-interpolate | 3.0.1 | ISC |
| d3-path | 3.1.0 | ISC |
| d3-scale | 4.0.2 | ISC |
| d3-shape | 3.2.0 | ISC |
| d3-time | 3.1.0 | ISC |
| d3-time-format | 4.1.0 | ISC |
| decimal.js-light | 2.5.1 | MIT |
| decode-named-character-reference | 1.3.0 | MIT |
| detect-node-es | 1.1.0 | MIT |
| devlop | 1.1.0 | MIT |
| es-toolkit | 1.52.0 | MIT |
| escape-string-regexp | 5.0.0 | MIT |
| estree-util-is-identifier-name | 3.0.0 | MIT |
| eventemitter3 | 5.0.4 | MIT |
| extend | 3.0.2 | MIT |
| file-selector | 5.0.1 | MIT |
| framer-motion | 14.0.0 | MIT |
| get-nonce | 1.0.1 | MIT |
| hast-util-to-jsx-runtime | 2.3.6 | MIT |
| hast-util-whitespace | 3.0.0 | MIT |
| html-url-attributes | 3.0.1 | MIT |
| immer | 10.2.0 | MIT |
| immer | 11.1.18 | MIT |
| inline-style-parser | 0.2.7 | MIT |
| internmap | 2.0.3 | ISC |
| is-plain-obj | 4.1.0 | MIT |
| longest-streak | 3.1.0 | MIT |
| lucide-react | 1.52.0 | ISC |
| markdown-table | 3.0.4 | MIT |
| mdast-util-find-and-replace | 3.0.2 | MIT |
| mdast-util-from-markdown | 2.0.3 | MIT |
| mdast-util-gfm | 3.1.0 | MIT |
| mdast-util-gfm-autolink-literal | 2.0.1 | MIT |
| mdast-util-gfm-footnote | 2.1.0 | MIT |
| mdast-util-gfm-strikethrough | 2.0.1 | MIT |
| mdast-util-gfm-table | 2.0.0 | MIT |
| mdast-util-gfm-task-list-item | 2.0.0 | MIT |
| mdast-util-phrasing | 4.1.0 | MIT |
| mdast-util-to-hast | 13.2.1 | MIT |
| mdast-util-to-markdown | 2.1.3 | MIT |
| mdast-util-to-string | 4.0.0 | MIT |
| micromark | 4.0.3 | MIT |
| micromark-core-commonmark | 2.0.4 | MIT |
| micromark-extension-gfm | 3.0.0 | MIT |
| micromark-extension-gfm-autolink-literal | 2.1.0 | MIT |
| micromark-extension-gfm-footnote | 2.1.0 | MIT |
| micromark-extension-gfm-strikethrough | 2.1.0 | MIT |
| micromark-extension-gfm-table | 2.1.2 | MIT |
| micromark-extension-gfm-tagfilter | 2.0.0 | MIT |
| micromark-extension-gfm-task-list-item | 2.1.0 | MIT |
| micromark-factory-destination | 2.0.1 | MIT |
| micromark-factory-label | 2.0.1 | MIT |
| micromark-factory-space | 2.1.0 | MIT |
| micromark-factory-title | 2.0.1 | MIT |
| micromark-factory-whitespace | 2.0.1 | MIT |
| micromark-util-character | 2.1.1 | MIT |
| micromark-util-chunked | 2.0.1 | MIT |
| micromark-util-classify-character | 2.0.1 | MIT |
| micromark-util-combine-extensions | 2.0.1 | MIT |
| micromark-util-decode-numeric-character-reference | 2.0.2 | MIT |
| micromark-util-decode-string | 2.0.1 | MIT |
| micromark-util-edit-map | 1.0.0 | MIT |
| micromark-util-encode | 2.0.1 | MIT |
| micromark-util-html-tag-name | 2.0.1 | MIT |
| micromark-util-normalize-identifier | 2.0.1 | MIT |
| micromark-util-resolve-all | 2.0.1 | MIT |
| micromark-util-sanitize-uri | 2.0.1 | MIT |
| micromark-util-subtokenize | 2.1.0 | MIT |
| motion | 14.0.0 | MIT |
| motion-dom | 14.0.0 | MIT |
| motion-utils | 14.0.0 | MIT |
| next-themes | 0.4.6 | MIT |
| property-information | 7.2.0 | MIT |
| radix-ui | 1.7.0 | MIT |
| react | 19.3.0 | MIT |
| react-dom | 19.3.0 | MIT |
| react-dropzone | 20.1.2 | MIT |
| react-is | 19.3.0 | MIT |
| react-markdown | 10.1.0 | MIT |
| react-redux | 9.3.0 | MIT |
| react-remove-scroll | 2.7.2 | MIT |
| react-remove-scroll-bar | 2.3.8 | MIT |
| react-router | 8.4.0 | MIT |
| react-style-singleton | 2.2.3 | MIT |
| recharts | 3.8.0 | MIT |
| redux | 5.0.1 | MIT |
| redux-thunk | 3.1.0 | MIT |
| remark-gfm | 4.0.1 | MIT |
| remark-parse | 11.0.0 | MIT |
| remark-rehype | 11.1.2 | MIT |
| reselect | 5.1.1 | MIT |
| reselect | 5.2.0 | MIT |
| rolldown | 1.2.11 | MIT |
| scheduler | 0.28.0 | MIT |
| shadcn | 4.21.4 | MIT |
| sonner | 2.0.8 | MIT |
| space-separated-tokens | 2.0.2 | MIT |
| style-mod | 4.1.4 | MIT |
| style-to-js | 1.1.21 | MIT |
| style-to-object | 1.0.14 | MIT |
| tailwindcss | 4.3.3 | MIT |
| tiny-invariant | 1.3.3 | MIT |
| trim-lines | 3.0.1 | MIT |
| trough | 2.2.0 | MIT |
| tslib | 2.8.1 | 0BSD |
| tw-animate-css | 1.4.0 | MIT |
| unified | 11.0.5 | MIT |
| unist-util-is | 6.0.1 | MIT |
| unist-util-position | 5.0.0 | MIT |
| unist-util-stringify-position | 4.0.0 | MIT |
| unist-util-visit | 5.1.0 | MIT |
| unist-util-visit-parents | 6.0.2 | MIT |
| uqr | 0.1.3 | MIT |
| use-callback-ref | 1.3.3 | MIT |
| use-sidecar | 1.1.3 | MIT |
| use-sync-external-store | 1.7.0 | MIT |
| vfile | 6.0.3 | MIT |
| vfile-message | 4.0.3 | MIT |
| victory-vendor | 37.3.6 | MIT AND ISC |
| vite | 8.3.3 | MIT |
| w3c-keyname | 2.2.8 | MIT |
| zwitch | 2.0.4 | MIT |

## NOTICE files of Go modules

The panel binary is built from Go modules; those with a NOTICE file (Apache License 2.0, section 4(d))
are reproduced here. The list follows `go version -m bin/kubedactyl`.

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
