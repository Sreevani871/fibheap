package xub

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	M "msg"
	S "server"
	"store/engine/redis"
	"util"
	"util/concurrency/pool"
	"xub/hooks"

	lru "github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	gometrics "github.com/rcrowley/go-metrics"
)

const (
	ONE_HOUR    = 3600 * 1000  // an hour in millis
	FOUR_HOUR   = 4 * ONE_HOUR //
	EIGHT_HOUR  = 8 * ONE_HOUR
	ONE_DAY     = 24 * ONE_HOUR // a day in millis
	BATCH_SIZE  = 500           // no of messages processed together
	MAX_WORKERS = 64            // no of workers
	WAIT_TIME   = 500           // 500 ms

	METRIC_WEBHOOKS = "WebHooks"
	// 1 - hsetnx and sadd pass
	// 0 - hsetnx fails
	// -1 - hsetnx pass and sadd fail => should never occur
	SAVE_MSG_SCRIPT = `
		local added = redis.call('HSETNX', KEYS[1], ARGV[1], ARGV[2])
		if added == 0 then
			redis.log(redis.LOG_VERBOSE, "failed to add to hmap")
			return 0
		end
		redis.log(redis.LOG_VERBOSE, "added to hmap")
		if redis.call('SADD', KEYS[2], ARGV[1]) == 1 then
			redis.log(redis.LOG_VERBOSE, "added to set")
			return 1
		end
		return -1
	`
)

var (
	webHookCount  int64
	webHookMetric gometrics.Gauge //No of Hooks
)

type Account interface {
	fmt.Stringer
	Id() int64
	Name() string
	Hash() string
	ToJSON() ([]byte, error)
	AuthKey() string
	SetAuthKey(string)
	RegisterHook(name, event, hookType, endPoint string) (string, error)
	HasHooks() bool
	loadHooks() (e error)
	getHookById(hashId string) (hooks.Hook, bool)
	PassMessage(*M.Message)
	Start(chan<- string)
	Stop(*sync.WaitGroup) error
	GetCounter() uint8
	IncrementCounter()
	IsRunning() bool
}

// Default Json encoder/decoder requires the struct and its fields to be public
// so an alibi struct
type JsonAccount struct {
	Id      int64  `json:"id,string"` // system provided
	HashId  string `json:"hash"`      // system provided
	Name    string `json:"name"`      //
	Version string `json:"version"`   //
	Desc    string `json:"desc"`      //
	AuthKey string `json:"key"`
}

type account struct {
	sync.RWMutex
	id         int64                          // system provided
	hashId     string                         // system provided
	name       string                         //
	version    string                         //
	desc       string                         //
	authKey    string                         // system provided
	evtHook    map[hooks.EventType]hooks.Hook // map of event => webHook. Only one hook per event allowed.
	channelMap *lru.Cache                     // channel hash => channel string
	pool       pool.WorkerPool                //
	msgArray   [BATCH_SIZE]*M.Message         // Array of messages for this account, received from accountmanager.
	staging    []*M.Message
	msgPipe    chan *M.Message //
	quit       chan struct{}   //
	qscav      chan struct{}
	counter    uint8 // retry count to track load hooks
	running    bool  // account start called
	scavenger  Scavenger
}

func init() {
	webHookCount = 0
	webHookMetric = gometrics.NewGauge()
	gometrics.Register(METRIC_WEBHOOKS, webHookMetric)
}

func (m *accountManager) newAccount(id int64, hashId, name, version, desc string) (Account, error) {
	authKey, err := util.GenerateRandomString(32)
	if err != nil {
		return nil, err
	}
	a := &account{
		id:         id,
		hashId:     hashId,
		name:       name,
		version:    version,
		desc:       desc,
		evtHook:    make(map[hooks.EventType]hooks.Hook, 4),
		msgPipe:    make(chan *M.Message, 1024),
		pool:       pool.NewPool("ACCOUNT-WORKER", MAX_WORKERS, 0),
		channelMap: lru.New(lru.Configure().MaxSize(10240).ItemsToPrune(512)),
		quit:       make(chan struct{}, 2),
		authKey:    authKey,
	}
	a.staging = a.msgArray[0:0]
	return a, nil
}

func ParseAccountJson(jsonByteOrString interface{}) (Account, error) {
	var (
		j JsonAccount
		e error
	)
	switch v := jsonByteOrString.(type) {
	case string:
		e = json.Unmarshal([]byte(jsonByteOrString.(string)), &j)
	case []byte:
		e = json.Unmarshal(jsonByteOrString.([]byte), &j)
	default:
		e = errors.New(fmt.Sprintf("Error parsing JSON msg. Unknown [Type: %T]", v))
		return nil, e
	}
	fmt.Printf("Parsed [JsonAccount: %v]", j)

	b := account{
		id:         j.Id,
		hashId:     j.HashId,
		name:       j.Name,
		version:    j.Version,
		desc:       j.Desc,
		authKey:    j.AuthKey,
		evtHook:    make(map[hooks.EventType]hooks.Hook, 4),
		msgPipe:    make(chan *M.Message, 1024),
		pool:       pool.NewPool("ACCOUNTMAX_WORKERS", MAX_WORKERS, 0),
		channelMap: lru.New(lru.Configure().MaxSize(10240).ItemsToPrune(512)),
		quit:       make(chan struct{}, 2),
		qscav:      make(chan struct{}, 1),
	}
	b.staging = b.msgArray[0:0]
	return &b, e
}

func (a *account) ToJSON() ([]byte, error) {
	var j JsonAccount
	j = JsonAccount{
		Id:      a.id,
		HashId:  a.hashId,
		Name:    a.name,
		Version: a.version,
		Desc:    a.desc,
		AuthKey: a.authKey,
	}

	jbArr, e := json.Marshal(j)
	log.Infof("Account in json: %s", jbArr)
	return jbArr, e
}

func (a *account) Name() string             { return a.name }
func (a *account) Id() int64                { return a.id }
func (a *account) Hash() string             { return a.hashId }
func (a *account) PassMessage(m *M.Message) { a.msgPipe <- m }
func (a *account) GetCounter() uint8        { return a.counter }
func (a *account) IncrementCounter()        { a.counter++ }
func (a *account) AuthKey() string          { return a.authKey }
func (a *account) SetAuthKey(k string)      { a.authKey = k }

func (a *account) HasHooks() bool {
	a.RLock()
	evtHookLength := len(a.evtHook)
	a.RUnlock()
	return evtHookLength > 0
}

func (a *account) IsRunning() bool {
	a.RLock()
	isRunning := a.running
	a.RUnlock()
	return isRunning
}

func (a *account) String() string {
	return fmt.Sprintf("[Account: %d %s %s]", a.id, a.name, a.desc)
}

func (a *account) Start(stopped chan<- string) {
	var (
		m           *M.Message
		batchTicker *time.Ticker = time.NewTicker(time.Millisecond * WAIT_TIME)
	)

	defer func() {
		if r := recover(); r != nil {
			log.Infof("Recovered from panic: %v. [Account: %s %s] ", r, a.name, a.hashId)
			debug.PrintStack()
		}

		batchTicker.Stop()
		log.Infof("Exiting [Account: %s %s] goroutine.", a.name, a.hashId)
		stopped <- a.hashId
	}()

	log.Infof("Starting [Account: %s] service...", a.Hash())
	a.Lock()
	a.running = true
	a.Unlock()

	_, err := a.getHookByEvent(hooks.EVT_NEW_MESSAGE.String())
	if err != nil {
		log.Errorf("acc:start failed to get hook acchash[%s] err[%s]", a.hashId, err)
	} else {
		log.Infof("acc:start: starting scavenger acchash[%s]", a.hashId)
		scavenger := newScavenger(a)
		a.scavenger = scavenger
		go a.scavenger.Start(a.qscav)
	}

	for {
		select {
		// Get messages from worker.
		case m = <-a.msgPipe:
			// log.Info(m)
			a.staging = append(a.staging, m)
			if len(a.staging) == BATCH_SIZE {
				a.batchSave()
			}

		case <-batchTicker.C: // time.After(time.Millisecond * WAIT_TIME):
			if ssize := len(a.staging); ssize > 0 {
				log.Debugf("TIMEOUT. save. [Stage size: %d]", ssize)
				a.batchSave()
			}

		case <-a.qscav:
			log.Infof("scavenger quit: restarting acchash[%s]", a.hashId)
			go a.scavenger.Start(a.qscav)

		case <-a.quit:
			if ssize := len(a.staging); ssize > 0 {
				log.Infof("Quitting. save. [Stage size: %d]", ssize)
				a.batchSave()
			}

			log.Infof("Shutdown called. [Account: %s %s] Main thread exiting...", a.name, a.hashId)
			time.Sleep(time.Millisecond * 512)
			return
		}
	}
}

func (a *account) Stop(wg *sync.WaitGroup) error {
	a.Lock()
	a.running = false
	a.Unlock()
	a.quit <- struct{}{} // quit scavenger
	if wg != nil {
		if a.scavenger != nil {
			wg.Add(1)
			a.scavenger.Stop(wg)
		}
		wg.Done()
	}
	return nil
}

// RegisterHook registers a hook to the account. It returns hook's hashid.
func (a *account) RegisterHook(name, event, hookType, endPoint string) (string, error) {
	var (
		hook      hooks.Hook
		hookEvent hooks.EventType
		e         error
	)
	log.Infof("Register [Hook: %s %s %s %s]", name, event, hookType, endPoint)

	// check event
	if hookEvent, e = hooks.ValidateEvent(event); e == nil {

		// check hook type
		switch strings.ToUpper(hookType) {
		case "WEBHOOK", "WEB_HOOK", "WEB HOOK":

			if id, e := generateHookId(); e == nil { // get Hook Id from redis

				// Create Hook
				if hook, e = hooks.NewWebHook(id, hookEvent, name, endPoint); e == nil {

					// save hook details & add hook to account
					if e = a.addHook(hook); e == nil {
						// return hook hash as hook id

						webHookCount += 1
						webHookMetric.Update(webHookCount)

						log.Infof("New Web[Hook: %s %d %s]", name, id, hook.HashId())

						return hook.HashId(), nil

					} else {
						s := fmt.Sprintf("Saving new web [Hook: %s]. [Error: %s]", name, e.Error())
						log.Errorf("%s", s)
						return "", errors.New(s)
					}

				} else {
					s := fmt.Sprintf("Creating new web [Hook: %s]. [Error: %s]", name, e.Error())
					log.Errorf("%s", s)
					return "", errors.New(s)
				}

			} else {
				s := fmt.Sprintf("Registering new [Hook: %s]. [Error: %s]", name, e.Error())
				log.Errorf("%s", s)
				return "", errors.New(s)
			}

		// case "REDIS_HOOK", "REDISHOOK": hooks.NewRedisHook()
		default:
			s := fmt.Sprintf("Invalid request. Wrong [Hook type: %s] passed.", hookType)
			log.Errorf("%s", s)
			return "", errors.New(s)
		}

	} else { // wrong event type
		return "", e
	}
}

func (a *account) addHook(h hooks.Hook) (e error) {
	var tries int = 5
	e = a.saveHookInfo(h)
	for i := 1; i <= tries; i++ {
		if e == nil {
			a.Lock()
			a.evtHook[h.Event()] = h
			a.Unlock()
			return nil

		} else {
			log.Errorf("Saving [Hook: %s]. [Error: %s]. [Retry: %d]", h.Name(), e.Error(), i)
			time.Sleep(time.Millisecond * time.Duration(i*i) * 10) // 10,40,90,160,250
			e = a.saveHookInfo(h)
		}
	}
	return e
}

func (a *account) removeHook(h hooks.Hook) (e error) {
	var tries int = 5
	e = a.deleteHook(h)
	for i := 1; i <= tries; i++ {
		if e == nil {
			a.Lock()
			delete(a.evtHook, h.Event())
			a.Unlock()

			// TODO: notify all other Xub's about deleted hook
			return nil

		} else {
			log.Errorf("Deleting [Hook %d %s %s]. [Error: %s]. [Retry: %d]",
				h.Id(), h.HashId(), h.Name(), e.Error(), i)
			time.Sleep(time.Millisecond * time.Duration(i*i) * 10) // 10,40,90,160,250
			e = a.deleteHook(h)
		}
	}
	return e
}

func (a *account) getHookByEvent(event string) (hooks.Hook, error) {
	if evtTyp, err := hooks.ValidateEvent(event); err == nil {
		a.RLock()
		hook, ok := a.evtHook[evtTyp]
		a.RUnlock()
		if ok {
			return hook, nil
		} else {
			return nil, errors.New(fmt.Sprintf("Hook not found for specified [Event: %s]", event))
		}
	} else {
		return nil, err
	}
}

func (a *account) getHookById(hashId string) (hooks.Hook, bool) {
	a.RLock()
	defer a.RUnlock()
	for _, h := range a.evtHook {
		if h.HashId() == hashId {
			return h, true
		}
	}
	return nil, false
}

func (a *account) loadHooks() (e error) {
	var (
		h               hooks.Hook
		hookIdArr       = make([]int64, 0, 4)
		keyAccountHooks = redis.REDIS_KEY_ACCOUNT_HOOKS + a.hashId
	)

	rcon := redisClient.GetConnection()
	defer rcon.Close()

	// Fetch hook ids. from Account Hooks Map from redis. Account event => Hook hashId
	rcon.Send("HVALS", keyAccountHooks)

	rcon.Flush()

	if v, er := rcon.Receive(); er == nil { // TODO: Use Lua script
		log.Infof("Loading Hook ids of [Account: %d %s] >> REDIS HVALS [%s] >> [%s]", a.Id(), a.Name(), keyAccountHooks, v.([]interface{}))
		arr := v.([]interface{})
		asize := len(arr)

		if asize > 0 { // if has hooks get hook details
			vargs := make([]interface{}, 0, asize+2)     // redis args
			vargs = append(vargs, redis.REDIS_KEY_HOOKS) // start with redis key

			for _, val := range arr {
				// log.Info("value == ", val.([]byte))
				hookHash := string(val.([]byte))
				hId := S.GetHookId(hookHash)
				hookIdArr = append(hookIdArr, hId)
				vargs = append(vargs, hId)
				// log.Trace(hookHash, hId)
			}
			log.Info(vargs, hookIdArr)

			// get Hook data
			rcon.Send("HMGET", vargs...) // send redis command and args

			rcon.Flush()

			if v, er = rcon.Receive(); er == nil {
				log.Infof("Hook data of [Account: %d %s]. REDIS HMGET [Key(s): %s args] >> [Reply: %s]", a.Id(), a.Name(), redis.REDIS_KEY_HOOKS, v.([]interface{}))

				arr = v.([]interface{})

				for _, byteArr := range arr {
					log.Infof("TYPE:%T %s", byteArr, byteArr)
					if h, er = hooks.ParseHookJson(byteArr); er == nil {

						// Set the event hook map
						a.Lock()
						a.evtHook[h.Event()] = h
						a.Unlock()
					} else {
						s := fmt.Sprintf("Err Parsing Hook JSON of [Account: %d %s]. [Data: %s] from REDIS. [Error: %s]",
							a.Id(), a.Name(), byteArr, er)
						log.Errorf("%s", s)
					}
				}
			} else {
				log.Errorf("Err receiving [Account: %d %s] REDIS HMGET [Key(s): %s args] >> [Error: %s]",
					a.Id(), a.Name(), redis.REDIS_KEY_HOOKS, er)

				if strings.Contains(er.Error(), "connection refused") {
					e = errors.New(fmt.Sprintf("Connection [Error: %s] for [Account: %d %s] REDIS HMGET [Key(s): %s args]",
						er, a.Id(), a.Name(), redis.REDIS_KEY_HOOKS))
				} else {
					e = er
				}
			}
		}
	} else {
		log.Errorf("Err [Account: %d %s] REDIS HVALS [Key: %s] >> [Error: %s]", a.Id(), a.Name(), keyAccountHooks, er.Error())
		if strings.Contains(er.Error(), "connection refused") {
			e = errors.New(fmt.Sprintf("Connection [Error: %s]. [Account: %d %s] REDIS HVALS [Key: %s] >> ",
				er, a.Id(), a.Name(), keyAccountHooks))
		} else {
			e = er
		}
	}

	return e
}

const (
	// Redis script to set a Hook against an account event
	// KEY-1 = REDIS_KEY_HOOKS = "PUBSUB_HOOKS"
	// KEY-2 = keyAccountHooks = REDIS_KEY_ACCOUNT_HOOKS + a.hashId
	// KEY-3 = hookId
	// KEY-4 = event
	// KEY-5 = hookHash
	// KEY-6 = value
	saveHookScript string = `
        local hash = ""
        redis.call("HSET", KEYS[1], KEYS[3], KEYS[6])
        if redis.call("HEXISTS", KEYS[2], KEYS[4]) == 0 then
            redis.call("HSET", KEYS[2], KEYS[4], KEYS[5])
        else
            hash = redis.call("HGET", KEYS[2], KEYS[4])
            redis.call("HSET", KEYS[2], KEYS[4], KEYS[5])
        end
        return hash
    ` // Lua Script
)

func (a *account) saveHookInfo(h hooks.Hook) (e error) {
	log.Infof("Saving [Hook: %s %d %s]", h.Name(), h.Id(), h.HashId())
	if jsonByteArr, e := h.ToJSON(); e == nil {
		log.Infof("Hook in [JSON: %s]", jsonByteArr)

		rcon := redisClient.GetConnection()
		defer rcon.Close()

		// TODO: Transaction
		// set Hooks Map in redis. HookId => Hook json data
		rcon.Send("HSET", redis.REDIS_KEY_HOOKS, h.Id(), jsonByteArr)

		// set Account Hooks Map in redis. Account event => Hook hashId
		keyAccountHooks := redis.REDIS_KEY_ACCOUNT_HOOKS + a.hashId
		rcon.Send("HSET", keyAccountHooks, h.Event().String(), h.HashId())
		// TODO: if overwriting, delete the previous hook. Use Lua Script

		rcon.Flush()

		if v, er := rcon.Receive(); er == nil {
			log.Infof("Successfully set Hook data. Recieved [Reply: %v]", v)
		} else {
			log.Errorf("REDIS HSET [Key(s): %s %d %s] >> [Error: %s]", redis.REDIS_KEY_HOOKS, h.Id(), jsonByteArr, er.Error())
			if strings.Contains(er.Error(), "connection refused") {
				e = errors.New(fmt.Sprintf("REDIS HSET [Key(s): %s %d %s] >> Connection [Error: %s]", redis.REDIS_KEY_HOOKS, h.Id(), jsonByteArr, er.Error()))
			} else {
				e = er
			}
		}

		if v, er := rcon.Receive(); er == nil {
			log.Infof("Successfully set Hook for account event. Recieved [Reply: %v]", v)
		} else {
			log.Errorf("REDIS HSET [Key(s): %s %s %s] >> [Error: %s]", keyAccountHooks, h.Event().String(), h.HashId(), er.Error())
			if strings.Contains(er.Error(), "connection refused") {
				e = errors.New(fmt.Sprintf("REDIS HSET [Key(s): %s %s %s] >> Connection [Error: %s]", keyAccountHooks, h.Event().String(), h.HashId(), er.Error()))
			} else {
				e = er
			}
		}

		return e
	} else {
		log.Errorf("Converting [Hook: %s %s] to JSON byte array. [Error: %s]", h.Name(), h.HashId(), e.Error())
		return errors.New(fmt.Sprintf("Converting [Hook: %s %s] to JSON byte array. [Error: %s]", h.Name(), h.HashId(), e.Error()))
	}
}

func (a *account) deleteHook(h hooks.Hook) (e error) {
	log.Infof("Delete Hook: %s %d %s", h.Name(), h.Id(), h.HashId())

	rcon := redisClient.GetConnection()
	defer rcon.Close()

	// TODO: Transaction
	// delete Hooks Map enrty in redis. HookId => Hook json data
	rcon.Send("HDEL", redis.REDIS_KEY_HOOKS, h.Id())

	// set Account Hooks Map entry in redis. Account event => Hook hashId
	keyAccountHooks := redis.REDIS_KEY_ACCOUNT_HOOKS + a.hashId
	rcon.Send("HDEL", keyAccountHooks, h.Event().String())

	rcon.Flush()

	if v, er := rcon.Receive(); er == nil {
		log.Infof("Successfully deleted Hook data. Recieved [Reply: %v]", v)
	} else {
		log.Errorf("REDIS HDEL [Key(s): %s %d] >> [Error: %s]", redis.REDIS_KEY_HOOKS, h.Id(), er.Error())
		if strings.Contains(er.Error(), "connection refused") {
			e = errors.New(fmt.Sprintf("REDIS HDEL [Key(s): %s %d] >> Connection [Error: %s]", redis.REDIS_KEY_HOOKS, h.Id(), er.Error()))
		} else {
			e = er
		}
	}

	if v, er := rcon.Receive(); er == nil {
		log.Info("Successfully deleted Hook for account event. Recieved [Reply: %v]", v)

	} else {
		log.Errorf("REDIS HDEL [Key(s): %s %s] >> [Error: %s]", keyAccountHooks, h.Event().String(), er.Error())
		if strings.Contains(er.Error(), "connection refused") {
			e = errors.New(fmt.Sprintf("REDIS HDEL [Key(s): %s %s] >> Connection [Error: %s]", keyAccountHooks, h.Event().String(), er.Error()))
		} else {
			e = er
		}
	}

	return e
}

// this is a synchronous call.
// no sync required here, as this portion will be handled by a single thread
func (a *account) batchSave() {
	if size := len(a.staging); size > 0 {
		a.RLock()
		hook := a.evtHook[hooks.EVT_NEW_MESSAGE]
		a.RUnlock()

		if hook != nil {
			log.Infof("Account batchSave. [Size: %d]", size)
			batch := make([]*M.Message, size)
			copy(batch, a.staging)
			a.staging = a.msgArray[0:0]
			w := a.pool.GetWorker()
			// delegate redis write to a concurrent worker
			// worker will return immediately after, spawning the task.
			// the worker should not be used for another task again.
			w.DoTask(callHook, batch, hook, a.hashId)

		} else {
			log.Info("Hook is empty. Quitting batchSave. Nothing to do.")
		}
	}
}

func callHook(vargs ...interface{}) {
	if len(vargs) != 3 {
		log.Error("callHook: invalid arguments : ")
		log.Info(vargs...)
		return
	}

	var (
		batch       []*M.Message = vargs[0].([]*M.Message)
		hook        hooks.Hook   = vargs[1].(hooks.Hook)
		accountHash string       = vargs[2].(string)
		uniqBatch   []M.ClientMessage
		uniqIds     []uint64
		dups        = make([]uint64, 0)
	)

	uniqBatch, uniqIds, dups, _ = deDuplicate(batch, accountHash)

	if uniqBatch == nil || uniqIds == nil {
		log.Infof("callHook: deduplicate returned no messages")
		return
	}
	if len(dups) > 0 {
		log.Infof("DeDuplicated [nos: %s] messages of [Accounthash: %s]. ", dups, accountHash)
	}
	log.Infof("Unique [IDs: %d]", uniqIds)
	// Web Hook call
	if e := hook.Write(uniqBatch, accountHash); e == nil {
		removeIdsFromRedis(uniqIds, accountHash)
	} else {
		log.Errorf("Err writing messages by [Hook: %s]. [Error: %s]", hook.Name(), e.Error())
		// leave it. Scavenger will take care. Try to resend
	}
}

// de-duplicate from redis
// Messages are written to redis hashmap keyed at redis.DEDUP_ACCOUNT_MSGS for acc in context.
// Msgs are also written to set (in redis) keyed at redis.REDIS_KEY_SND_MSG_OF_ACCOUNT.
// When scavenger wakes up, msg ids are read from set and hashmap is refered to get msg body.
func deDuplicate(messages []*M.Message, accHash string) ([]M.ClientMessage, []uint64, []uint64, error) {
	var (
		size     = len(messages)
		uniqMsgs = make([]M.ClientMessage, 0, size)
		uniqIds  = make([]uint64, 0, size)
		dedup    = make([]uint64, 0)
		hr, nhr  int
	)

	rconn := redisClient.GetConnection()
	defer rconn.Close()
	sc := redisClient.NewScript(2, SAVE_MSG_SCRIPT)
	setkey := redis.REDIS_KEY_SND_MSG_OF_ACCOUNT + accHash
	hr = S.GetHourOfTheDayFrom(uint64(messages[0].Id))
	for i := 0; i < size; i++ {
		id := strconv.FormatUint(uint64(messages[i].Id), 10)
		body, err := json.Marshal(messages[i])
		if err != nil {
			log.Errorf("DeDuplicate [Accounthash: %s]. Marshaling [Message: %v]. [Error: %s]", accHash, messages[i], err.Error())
			continue
		}
		nhr = S.GetHourOfTheDayFrom(uint64(messages[i].Id))
		dedupAndSaveKey := fmt.Sprintf(redis.DEDUP_ACCOUNT_MSGS, accHash, nhr)
		sc.Send(rconn, dedupAndSaveKey, setkey, id, body)
	}
	if err := rconn.Flush(); err != nil {
		log.Errorf("DeDuplicate [Accounthash: %s]. Flush failed. [Error: %s]", accHash, err.Error())
		return nil, nil, nil, errors.Wrap(err, "failed to flush")
	}
	for i := 0; i < size; i++ {
		added, err := redis.Int(rconn.Receive())
		if err != nil {
			log.Errorf("DeDuplicate [Accounthash: %s]. Receiving. [Error: %s]", accHash, err.Error())
			continue
		}
		m := messages[i]
		if added == 1 {
			log.Infof("DeDuplicate [Accounthash: %s]. Added [Msg: %d]", accHash, m.Id)
			cm := m.GetCM()
			uniqMsgs = append(uniqMsgs, cm)
			uniqIds = append(uniqIds, uint64(m.Id))
		} else if added == 0 {
			dedup = append(dedup, uint64(m.Id))
		} else {
			log.Errorf("DeDuplicate [Accounthash: %s] . hsetnx pass and sadd fail", accHash)
		}
	}
	if hr < nhr {
		hkey := fmt.Sprintf(redis.DEDUP_ACCOUNT_MSGS, accHash, nhr)
		log.Infof("DeDuplicate [Accounthash: %s] . Setting expiry for hmap [Key: %s]", accHash, hkey)
		if set, err := redisClient.Expire(hkey, EIGHT_HOUR); err != nil {
			log.Errorf("DeDuplicate [Accounthash: %s] . Failed to set expiry [Key: %s]. [Error: %s]", accHash, hkey, err.Error())
		} else if set == 1 {
			log.Infof("DeDuplicate [Accounthash: %s] . Expiry set [Key: %s]", accHash, hkey)
		} else {
			log.Errorf("DeDuplicate [Accounthash: %s] . Expiry failed, [Key: %s] does not exist", accHash, hkey)
		}
	}
	return uniqMsgs, uniqIds, dedup, nil
}

// remove from send set
func removeIdsFromRedis(uniqIds []uint64, accountId string) {
	var (
		size  = len(uniqIds)
		key   = redis.REDIS_KEY_SND_MSG_OF_ACCOUNT + accountId
		args  = make([]interface{}, 0, size)
		err   error
		reply interface{}
	)
	// log.Debug("Removing Id's from redis. ", uniqIds)

	args = append(args, key)
	for _, v := range uniqIds {
		args = append(args, v)
	}

	// Get redis connection
	rcon := redisClient.GetConnection()
	defer rcon.Close()

	if err = rcon.Send("SREM", args...); err == nil {
		if err = rcon.Flush(); err == nil {
			if reply, err = rcon.Receive(); err == nil {
				log.Infof("Recieved [Reply: %v]. Removed [Ids: %d] [uniqIds: %#v]", reply, len(uniqIds), uniqIds)
			} else {
				log.Errorf("Could not remove [Ids: %d]. [Error: %s]", uniqIds, err.Error())
			}
		} else {
			log.Errorf("Could not remove [Ids: %d]. [Error: %s]", uniqIds, err.Error())
		}
	} else {
		log.Errorf("Could not remove [Ids: %d]. [Error: %s]", uniqIds, err.Error())
	}
}
