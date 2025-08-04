package blocklogger

const (
	BLOCK_ENTER_NEW_ROUND                = "enter-new-round"
	BLOCK_EXIT_ENTER_NEW_ROUND           = "exit-enter-new-round"
	BLOCK_ENTER_PROPOSE                  = "enter-propose"
	BLOCK_EXIT_ENTER_PROPOSE             = "exit-enter-propose"
	BLOCK_ENTER_PRECOMMIT                = "enter-precommit"
	BLOCK_EXIT_ENTER_PRECOMMIT           = "exit-enter-precommit"
	BLOCK_ENTER_PREVOTE                  = "enter-prevote"
	BLOCK_EXIT_ENTER_PREVOTE             = "eixt-enter-prevote"
	BLOCK_ENTER_PRECOMMIT_WAIT           = "enter-precommit-wait"
	BLOCK_EXIT_ENTER_PRECOMMIT_WAIT      = "exit-enter-precommit-wait"
	BLOCK_ENTER_PREVOTE_WAIT             = "enter-prevote-wait"
	BLOCK_EXIT_ENTER_PREVOTE_WAIT        = "eixt-enter-prevote-wait"
	BLOCK_ENTER_CROSSSHARD_ACCEPT        = "enter-CSA"
	BLOCK_ENTER_CROSSSHARD_PROPOSAL      = "enter-CSP"
	BLOCK_ENTER_CROSSSHARD_COMMIT        = "enter-CSC"
	BLOCK_EXIT_ENTER_CROSSSHARD_ACCEPT   = "exit-enter-CSA"
	BLOCK_EXIT_ENTER_CROSSSHARD_PROPOSAL = "exit-enter-CSP"
	BLOCK_EXIT_ENTER_CROSSSHARD_COMMIT   = "exit-enter-CSC"
	BLOCK_FINALIZE_COMMIT                = "finalize-commit"
	BLOCK_FINISH_HEIGHT                  = "finish"
)

const (
	// 消息类型
	BLOCK_GET_PREVOTE             = "get-prevote"
	BLOCK_GET_PRECOMMIT           = "get-precommit"
	BLOCK_GET_CROSSSHARD_PART     = "get-crossshard-part"
	BLOCK_GET_CROSSSHARD_COMMIT   = "get-crossshard-commit"
	BLOCK_GET_CROSSSHARD_PROPOSAL = "get-crossshard-proposal"
	BLOCK_GET_PROPOSAL            = "get-proposal"
	BLOCK_GET_PART                = "get-part"
)
