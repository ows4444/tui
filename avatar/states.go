package avatar

// The animation, in one picture.
//
// A Model is in three independent machines at once. Each ignores the ticks of
// another, because every tick carries the token (owner or poseOwner) of the
// animation that scheduled it, and a new animation replaces its token.
//
// Eyes (owner, blink, idling, beat):
//
//	rest --Blink()--------------> blinking --tick x len(blinkOpen)--> rest
//	rest --StartIdle()/Hover(true)--> idling --tick--> idling (a beat: breathe,
//	      glance aside, and every blinkEvery beats start a blink)
//	idling --blink frames done--> idling
//	idling --StopIdle()/Hover(false)--> rest
//	blinking --Blink()--> blinking (restarts at frame 1; old ticks are dropped)
//	any --motion.Reduced--> rest (Blink, StartIdle and React return nil)
//
// Pose (from, tween, poseOwner, Expression, reaction):
//
//	still --SetExpression(e)--> easing (tween 1 to tweenFrames, one frame per tick)
//	easing --last tick, pose does not move--> still
//	easing --last tick, pose trembles or rocks--> moving
//	moving --tick--> moving (wobble cycles through wobbleCycle frames)
//	moving --SetExpression(still pose)--> easing
//
// Reaction (reaction, reacts, holdLeft), on top of the pose machine:
//
//	still --React()--> easing into the reaction
//	easing (reaction) --last tick--> held: a release timer, or for a moving pose
//	      holdLeft counts wobble frames
//	held --releaseMsg / holdLeft = 0--> easing back to Expression --> still
//	held --React()--> easing into another reaction (the hold starts again)
//	held --SetExpression(e)--> held (the change shows at release)
//
// Look (LookX, LookY, X, Y): not animated. LookAt and the mouse set it, Update
// follows MouseEvents through Bounds, and Hover(true/false) drives the idle
// machine above.
