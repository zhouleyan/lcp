import type { Messages } from "../../types"
import common from "./common"
import iam from "./iam"
import dashboard from "./dashboard"
import audit from "./audit"
import infra from "./infra"
import network from "./network"

const enUS: Messages = {
  ...common,
  ...iam,
  ...dashboard,
  ...audit,
  ...infra,
  ...network,
}

export default enUS
