package controllers

import (
	"fmt"
	"math"
	"strings"
	"time"

	eos "github.com/eoscanada/eos-go"
	"github.com/kataras/iris"

	"icbc-walking-go/entities"
	"icbc-walking-go/misc"
	"icbc-walking-go/models"
)

type qGameEnter struct {
	Owner    eos.AccountName `json:"owner"`
	Quantity eos.Asset       `json:"quantity"`
}

type qGameWin struct {
	Owner    eos.AccountName `json:"owner"`
	Nonce    string          `json:"nonce"`
	MinBonus eos.Asset       `json:"min_bonus"`
}

type qGameBonusRespRow struct {
	Last eos.Asset `json:"last"`
}

func newQuizGameEnter(name string, count float64) (*eos.Action, error) {
	quantity, err := misc.NewAsset(count)
	if err != nil {
		return nil, err
	}
	memo := fmt.Sprintf("%s enter a quiz game, fee[%s] ts[%d]", name, quantity.String(), time.Now().UnixNano()/int64(math.Pow10(3)))
	misc.Logger.Infof("%s\n", memo)

	return &eos.Action{
		Account: eos.AN(misc.Conf.GetString("blockchain.xrunda_contract_account")),
		Name:    eos.ActN("qgameenter"),
		Authorization: []eos.PermissionLevel{
			{Actor: eos.AN(name), Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(qGameEnter{
			Owner:    eos.AN(name),
			Quantity: quantity,
		}),
	}, nil
}


func newQuizGameWin(name, nonce string) (*eos.Action, error) {
	quantity, err := misc.NewAsset(misc.Conf.GetFloat64("qgame.min_bonus"))
	if err != nil {
		return nil, err
	}

	memo := fmt.Sprintf("%s win a quiz game, min bonus[%s] ts[%d]", name, quantity.String(), time.Now().UnixNano()/int64(math.Pow10(3)))
	misc.Logger.Infof("%s\n", memo)

	from := misc.Conf.GetString("blockchain.question_game_account")
	return &eos.Action{
		Account: eos.AN(misc.Conf.GetString("blockchain.xrunda_contract_account")),
		Name:    eos.ActN("qgamewin"),
		Authorization: []eos.PermissionLevel{
			{Actor: eos.AN(from), Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(qGameWin{
			Owner:    eos.AN(name),
			Nonce:    nonce,
			MinBonus: quantity,
		}),
	}, nil
}

/*
func newQuesGameWin(to string, count float64) (*eos.Action, error) {
	quantity, err := misc.NewAsset(count)
	if err != nil {
		return nil, err
	}
	memo := fmt.Sprintf("%s win a question game, bonus[%s] ts[%d]", to, quantity.String(), time.Now().UnixNano()/int64(math.Pow10(3)))
	misc.Logger.Infof("%s\n", memo)

	from := misc.Conf.GetString("blockchain.question_game_account")
	return &eos.Action{
		Account: eos.AN(misc.Conf.GetString("blockchain.contract_account")),
		Name:    eos.ActN("transfer"),
		Authorization: []eos.PermissionLevel{
			{Actor: eos.AN(from), Permission: eos.PN("active")},
		},
		ActionData: eos.NewActionData(token.Transfer{
			From:     eos.AN(from),
			To:       eos.AN(to),
			Quantity: quantity,
			Memo:     memo,
		}),
	}, nil
}
*/

// HandleQuizGameEnter handle POST /api/quiz/enter
func HandleQuizGameEnter(ctx iris.Context) (interface{}, *appError) {
	qGameEnter := &entities.QGameEnter{}
	if err := ctx.ReadJSON(qGameEnter); err != nil {
		return nil, &appError{err, "invalid post body", iris.StatusBadRequest}
	}
	/*
		remainUno, appErr := consumeUno(
			misc.Conf.GetString("blockchain.question_game_account"),
			qgameEnter.UserID,
			int16(misc.Conf.GetInt("server.qgame_type")),
			qgameEnter.Count,
		)
		if appErr.code != iris.StatusOK {
			return nil, appErr
		}
		misc.QGameEnter(qgameEnter.Count)
	*/
	userID := qGameEnter.UserID
	count := qGameEnter.Count
	eosName := nameToString(userID)
	trans, err := newQuizGameEnter(eosName, count)
	if err != nil {
		return nil, &appError{err, "new action for bc failed", iris.StatusInternalServerError}
	}
	txID, txData, err := misc.ToBlockchain([]*eos.Action{trans}, misc.Conf.GetString("blockchain.actor_key"))
	if err != nil {
		return nil, &appError{err, "to blockchain failed", iris.StatusInternalServerError}
	}

	// 保存到DB
	cType := int16(misc.Conf.GetInt("server.qgame_type"))
	models.SetConsumptionDB(txID, txData, userID, cType, count)

	// 异步检查交易是否已被确认, 如果OK了, 修改DB中的标记
	misc.Pool.JobQueue <- func() {
		misc.Pool.WaitCount(1)
		defer func() {
			misc.Pool.JobDone()
		}()
		misc.Logger.Debugf("begin to check consumption action in tx %s for %d(%s) when enter qgame, count[%f]\n", txID, userID, eosName, count)
		checkAndRetryConsumptionAction(txID, userID, cType, count, eosName, []*eos.Action{trans})
		misc.Logger.Debugf("check action in tx[%s] done\n", txID)
	}

	data := &entities.QGameEnterResponse{Consumed: count}
	if remainUno, err := queryUnoFromBC(eosName); err == nil {
		data.RemainUno = remainUno
	} else {
		misc.Logger.Warnf("query remain uno after entering qgame for %d(%s) failed, error[%v]\n", userID, eosName, err)
	}
	return data, &appError{err, "succeed", iris.StatusOK}
}

// HandleQuizGameWin handle POST /api/quiz/win
func HandleQuizGameWin(ctx iris.Context) (interface{}, *appError) {
	qgameWin := &entities.QGameWin{}
	if err := ctx.ReadJSON(qgameWin); err != nil {
		return nil, &appError{err, "invalid post body", iris.StatusBadRequest}
	}

	/*
		count := misc.QGameGo(qgameWin.UserID, qgameWin.Nonce)

		// TODO: need a restoration if failed
		// 调用智能合约来进行奖励
		bonusType := int16(misc.Conf.GetInt("server.qgame_type"))
		eosName := nameToString(qgameWin.UserID)
		trans, err := newQuesGameWin(eosName, count)
		if err != nil {
			return nil, &appError{err, "new action for bc failed", iris.StatusInternalServerError}
		}
		txID, err := misc.ToBlockchain([]*eos.Action{trans}, misc.Conf.GetString("blockchain.actor_key"))
		if err != nil {
			return nil, &appError{err, "to blockchain failed", iris.StatusInternalServerError}
		}

		// 保存到DB
		models.SetBonusDB(txID, qgameWin.UserID, bonusType, count)

		// 异步检查交易是否已被确认, 如果OK了, 修改DB中的标记
		misc.Pool.JobQueue <- func() {
			if nil == misc.CheckTransaction(txID) {
				models.SetBonusDBDone(txID)
				misc.Logger.Infof("%d got a bonus %f when win qgame\n", qgameWin.UserID, count)
			}
		}
	*/

	userID := qgameWin.UserID
	eosName := nameToString(userID)
	trans, err := newQuizGameWin(eosName, qgameWin.Nonce)
	if err != nil {
		return nil, &appError{err, "new action for bc failed", iris.StatusInternalServerError}
	}
	txID, txData, err := misc.ToBlockchain([]*eos.Action{trans}, misc.Conf.GetString("blockchain.actor_key"))
	if err != nil {
		if strings.Contains(err.Error(), "no quiz game pool exists") {
			return nil, &appError{err, "you should enter guiz game first", iris.StatusBadRequest}
		}
		return nil, &appError{err, "to blockchain failed", iris.StatusInternalServerError}
	}

	// 保存到DB
	bonusType := int16(misc.Conf.GetInt("server.qgame_type"))
	models.SetBonusDB(txID, txData, userID, bonusType, 0)

	// 异步检查交易是否已被确认, 如果OK了, 修改DB中的标记
	misc.Pool.JobQueue <- func() {
		misc.Pool.WaitCount(1)
		defer func() {
			misc.Pool.JobDone()
		}()
		// TODO: no count here, need update it after the query below
		misc.Logger.Debugf("begin to check bonus action in tx %s for %d(%s) when win qgame, count[%f]\n", txID, userID, eosName, float64(0))
		checkAndRetryBonusAction(txID, userID, bonusType, 0, eosName, []*eos.Action{trans})
		misc.Logger.Debugf("check action in tx[%s] done\n", txID)
	}

	data := &entities.QGameWinResponse{}
	if resp, err := misc.Bc.GetTableRows(eos.GetTableRowsRequest{
		Code:  misc.Conf.GetString("blockchain.xrunda_contract_account"),
		Scope: eosName,
		Table: "qgamebonus",
		JSON:  true,
	}); err == nil {
		qgameBonus := []qGameBonusRespRow{}
		if err1 := resp.JSONToStructs(&qgameBonus); err1 == nil {
			// TODO: 处理返回的更多行标记
			asset := qgameBonus[0].Last
			data.Bonus = float64(asset.Amount) / math.Pow10(int(asset.Precision))
		} else {
			misc.Logger.Warnf("parse the bonus response for %d(%s) failed when win qgame, error[%v] raw[%s]\n", userID, eosName, err1, resp.Rows)
		}
	} else {
		misc.Logger.Warnf("get table rows of %d(%s) failed when win qgame, error[%v]\n", userID, eosName, err)
	}
	if remainUno, err := queryUnoFromBC(eosName); err == nil {
		data.RemainUno = remainUno
	} else {
		misc.Logger.Warnf("query remain uno after winning qgame for %d(%s) failed, error[%v]\n", userID, eosName, err)
	}
	return data, &appError{err, "succeed", iris.StatusOK}
}

// HandleQuizBalance handle GET /api/quiz/balance
func HandleQuizBalance(ctx iris.Context) (interface{}, *appError) {
	balance, err := queryUnoFromBC(misc.Conf.GetString("blockchain.question_game_account"))
	if err != nil {
		return nil, &appError{err, "query quiz game balance failed", iris.StatusInternalServerError}
	}
	resp := &entities.QGameBalanceResponse{Balance: balance}
	if id, err := ctx.URLParamInt("user_id"); err == nil {
		eosName := nameToString(uint32(id))
		if uno, err := queryUnoFromBC(eosName); err == nil {
			resp.RemainUno = uno
		} else {
			misc.Logger.Warnf("query remain uno of %d(%s) failed when query quiz balance, error[%v]\n", id, eosName, err)
		}
	}
	return resp, &appError{nil, "succeed", iris.StatusOK}
}
