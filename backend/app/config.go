// Package app is what this app gives the base (backend/base) besides its routes and controllers: its names, its
// brand and the prefixes of its tokens and labels. Plain values, like frontend/src/app/config.ts.
package app

// Name is the product name in the start banner and the license file.
const Name = "Kubedactyl"

// ID names the binary, the chart and the image (`kubedactyl upgrade`, oci://…/charts/kubedactyl).
const ID = "kubedactyl"

// Domain prefixes the labels and annotations the panel sets (kubedactyl.io/upgrade-version).
const Domain = "kubedactyl.io"

// Tagline follows the version in the start banner.
const Tagline = "game servers on Kubernetes"

// BrandTagline is shown under the name in the sidebar and on the sign-in page unless the settings name another.
const BrandTagline = "Game servers on Kubernetes"

// EnvPrefix starts the names of the app's environment variables (KUBEDACTYL_NAMESPACE, KUBEDACTYL_ADMIN_PASSWORD).
const EnvPrefix = "KUBEDACTYL"

// APITokenPrefix marks personal access tokens ("kdt_<id>_<secret>").
const APITokenPrefix = "kdt_"

// CookiePrefix starts the names of the panel's cookies (kd_session, kd_device, kd_oidc).
const CookiePrefix = "kd_"

// KubeAPIUserQPS is how many requests per second one user may send to the panel when the settings name none.
const KubeAPIUserQPS = 10

// Logo is printed once when the panel starts (FIGlet font "smblock", one column between the letters).
const Logo = `▌ ▌ ▌ ▌ ▛▀▖ ▛▀▘ ▛▀▖ ▞▀▖ ▞▀▖ ▀▛▘ ▌ ▌ ▌
▙▞  ▌ ▌ ▙▄▘ ▙▄  ▌ ▌ ▙▄▌ ▌    ▌  ▝▞  ▌
▌▝▖ ▌ ▌ ▌ ▌ ▌   ▌ ▌ ▌ ▌ ▌ ▖  ▌   ▌  ▌
▘ ▘ ▝▀  ▀▀  ▀▀▘ ▀▀  ▘ ▘ ▝▀   ▘   ▘  ▀▀▘`

// BrandFrom and BrandTo are the brand gradient of the web interface (emerald-500 to sky-600) as RGB.
var BrandFrom, BrandTo = [3]int{0x00, 0xbc, 0x7d}, [3]int{0x00, 0x84, 0xd1}

// VersionModules are the Go modules of this app the settings page shows with their version, after the base ones
// (label and module path).
var VersionModules = [][2]string{}
