package types

type StrategyName string

const (
	StrategySkeleton            = StrategyName("skeleton")
	StrategyDataCollection      = StrategyName("datacollection")
	StrategyNaiveLiquidityMaker = StrategyName("naiveliquiditymaker")
	StrategyGrid                = StrategyName("grid")
	StrategyFlashCrash          = StrategyName("flashcrash")
	StrategyWashTradeTaker      = StrategyName("washtradetaker")
	StrategyNaivePriceTaker     = StrategyName("naivepricetaker")
	StrategyBollMaker           = StrategyName("bollmaker")
	StrategyBollMakerTPSL       = StrategyName("bollmakertpsl")
	StrategyXemm                = StrategyName("xemm")
	StrategyAvellanedaStoikov   = StrategyName("avellanedastoikov")
)
