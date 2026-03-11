import type { Messages } from "../../types"
import common from "./common"
import iam from "./iam"
import dashboard from "./dashboard"
import audit from "./audit"
import infra from "./infra"

const zhCN: Messages = {
  ...common,
  ...iam,
  ...dashboard,
  ...audit,
  ...infra,
}

export default zhCN
