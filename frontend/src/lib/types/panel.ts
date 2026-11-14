// The panel settings of this app: storage classes and load balancer pools to pick from. The base types (users,
// settings, upgrades) are in base/lib/types.ts.

export type {
  HttpapiPoolList as PoolList,
  HttpapiStorageClassList as StorageClassList,
  AppsettingsPool as Pool,
} from "@/lib/types/api.gen"
