import type { Messages } from "../../types"
import common from "./common"
import iam from "./iam"
import dashboard from "./dashboard"
import audit from "./audit"

const zhCN: Messages = {
  ...common,
  ...iam,
  ...dashboard,
  ...audit,
}

export default zhCN
