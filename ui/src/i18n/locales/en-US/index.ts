import type { Messages } from "../../types"
import common from "./common"
import iam from "./iam"
import dashboard from "./dashboard"
import audit from "./audit"
import infra from "./infra"
import network from "./network"
import o11y from "./o11y"
import pki from "./pki"

const enUS: Messages = {
  ...common,
  ...iam,
  ...dashboard,
  ...audit,
  ...infra,
  ...network,
  ...o11y,
  ...pki,
}

export default enUS
