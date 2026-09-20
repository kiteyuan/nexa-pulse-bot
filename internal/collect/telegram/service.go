package telegram

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
)

type LoginState = ports.LoginState

var _ ports.TelegramAdmin = (*Service)(nil)

type Service struct {
	DB       ports.TelegramRepo
	Sessions string
	Proxy    string
	Bed      ports.ImageUploader

	mu    sync.Mutex
	login map[int64]*LoginState
	stop  map[int64]context.CancelFunc
	gen   map[int64]uint64
}

func (s *Service) Name() string { return kernel.SourceTelegram }

func (s *Service) Sync(ctx context.Context, accountID int64) (int, error) {
	acc, err := s.DB.GetAccount(ctx, accountID)
	if err != nil {
		return 0, err
	}
	var n int
	err = s.run(ctx, acc, true, func(ctx context.Context, client *telegram.Client) error {
		st, err := client.Auth().Status(ctx)
		if err != nil {
			return err
		}
		if !st.Authorized {
			return fmt.Errorf("账号未登录，请先扫码")
		}
		api := client.API()
		dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})
		if err != nil {
			return err
		}
		for _, chat := range chats(dialogs) {
			ch, ok := chat.(*tg.Channel)
			if !ok || ch.Min {
				continue
			}
			// 只要广播频道；超级群 / 关联讨论组直接丢掉（含历史误入库）
			if !broadcastChannel(ch) {
				_ = s.DB.DeleteChannelByTelegram(ctx, accountID, ch.ID)
				continue
			}
			username, _ := ch.GetUsername()
			if err := s.DB.UpsertChannel(ctx, kernel.Channel{
				AccountID:  accountID,
				TelegramID: ch.ID,
				AccessHash: ch.AccessHash,
				Username:   username,
				Title:      ch.Title,
			}); err != nil {
				return err
			}
			n++
		}
		return s.DB.SetAccountStatus(ctx, accountID, "online")
	})
	if err != nil {
		return n, err
	}
	s.DB.AddLog(ctx, "INFO", "telegram", fmt.Sprintf("账号 #%d 同步频道 %d 个", accountID, n))
	return n, nil
}

func (s *Service) Poll(ctx context.Context) error {
	channels, err := s.DB.ActiveChannels(ctx)
	if err != nil || len(channels) == 0 {
		return err
	}
	byAccount := map[int64][]kernel.Channel{}
	for _, ch := range channels {
		byAccount[ch.AccountID] = append(byAccount[ch.AccountID], ch)
	}
	var total int
	for accountID, list := range byAccount {
		acc, err := s.DB.GetAccount(ctx, accountID)
		if err != nil {
			continue
		}
		err = s.run(ctx, acc, true, func(ctx context.Context, client *telegram.Client) error {
			st, err := client.Auth().Status(ctx)
			if err != nil {
				return err
			}
			if !st.Authorized {
				return nil
			}
			for _, ch := range list {
				n, err := s.pollOne(ctx, client.API(), ch)
				if err != nil {
					s.DB.AddLog(ctx, "ERROR", "telegram", fmt.Sprintf("轮询失败 %s: %v", ch.Title, err))
					continue
				}
				total += n
			}
			return nil
		})
		if err != nil {
			s.DB.AddLog(ctx, "ERROR", "telegram", err.Error())
		}
	}
	if total > 0 {
		s.DB.AddLog(ctx, "INFO", "telegram", fmt.Sprintf("本轮入库 %d 条", total))
	}
	return nil
}

func (s *Service) pollOne(ctx context.Context, api *tg.Client, ch kernel.Channel) (int, error) {
	peer := &tg.InputPeerChannel{ChannelID: ch.TelegramID, AccessHash: ch.AccessHash}
	if ch.LastMessageID <= 0 {
		hist, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{Peer: peer, Limit: 1})
		if err != nil {
			return 0, err
		}
		msgs := messages(hist)
		if len(msgs) > 0 {
			return 0, s.DB.SetChannelCursor(ctx, ch.ID, int64(msgs[0].ID))
		}
		return 0, nil
	}
	hist, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		MinID: int(ch.LastMessageID),
		Limit: 50,
	})
	if err != nil {
		return 0, err
	}
	msgs := messages(hist)
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].ID < msgs[j].ID })
	var ingested int
	maxID := ch.LastMessageID
	for _, msg := range msgs {
		if int64(msg.ID) > maxID {
			maxID = int64(msg.ID)
		}
		text := msg.Message
		paths, _ := s.savePhoto(ctx, api, msg)
		if text == "" && len(paths) == 0 {
			continue
		}
		hashKey := text
		if hashKey == "" {
			hashKey = fmt.Sprintf("media:%d:%d", ch.ID, msg.ID)
		}
		ok, err := s.DB.InsertRaw(ctx, kernel.RawItem{
			SourceKind:  kernel.SourceTelegram,
			SourceID:    ch.ID,
			ExternalID:  fmt.Sprintf("%d", msg.ID),
			Content:     text,
			ContentHash: kernel.Hash(hashKey),
			Media:       paths,
		})
		if err != nil {
			return ingested, err
		}
		if ok {
			ingested++
		}
	}
	if maxID > ch.LastMessageID {
		_ = s.DB.SetChannelCursor(ctx, ch.ID, maxID)
	}
	return ingested, nil
}

func (s *Service) savePhoto(ctx context.Context, api *tg.Client, msg *tg.Message) ([]string, error) {
	media, ok := msg.Media.(*tg.MessageMediaPhoto)
	if !ok || media.Photo == nil {
		return nil, nil
	}
	photo, ok := media.Photo.(*tg.Photo)
	if !ok {
		return nil, nil
	}
	if s.Bed == nil {
		s.DB.AddLog(ctx, "ERROR", "imgbed", "未配置图床，已跳过图片")
		return nil, nil
	}
	var buf bytes.Buffer
	loc := &tg.InputPhotoFileLocation{
		ID:            photo.ID,
		AccessHash:    photo.AccessHash,
		FileReference: photo.FileReference,
		ThumbSize:     largestThumb(photo),
	}
	if _, err := downloader.NewDownloader().Download(api, loc).Stream(ctx, &buf); err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, nil
	}
	ext, ok := kernel.SniffImageExt(buf.Bytes())
	if !ok {
		return nil, nil
	}
	link, err := s.Bed.Upload(ctx, fmt.Sprintf("%d%s", msg.ID, ext), buf.Bytes())
	if err != nil {
		s.DB.AddLog(ctx, "ERROR", "imgbed", "上传失败")
		return nil, nil
	}
	return []string{link}, nil
}

func largestThumb(photo *tg.Photo) string {
	best := "x"
	var area int
	for _, size := range photo.Sizes {
		if s, ok := size.(*tg.PhotoSize); ok && s.W*s.H > area {
			area = s.W * s.H
			best = s.Type
		}
	}
	return best
}

func chats(d tg.MessagesDialogsClass) []tg.ChatClass {
	switch v := d.(type) {
	case *tg.MessagesDialogs:
		return v.Chats
	case *tg.MessagesDialogsSlice:
		return v.Chats
	default:
		return nil
	}
}

func broadcastChannel(ch *tg.Channel) bool {
	return ch != nil && ch.Broadcast && !ch.Megagroup
}

func messages(h tg.MessagesMessagesClass) []*tg.Message {
	var raw []tg.MessageClass
	switch v := h.(type) {
	case *tg.MessagesMessages:
		raw = v.Messages
	case *tg.MessagesMessagesSlice:
		raw = v.Messages
	case *tg.MessagesChannelMessages:
		raw = v.Messages
	}
	var out []*tg.Message
	for _, m := range raw {
		if msg, ok := m.(*tg.Message); ok {
			out = append(out, msg)
		}
	}
	return out
}
