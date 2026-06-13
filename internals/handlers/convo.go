package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"recallo/internals/logger"
	"recallo/internals/middleware"
	"recallo/internals/models"
	"recallo/internals/utils"
)

// Pagination defaults for message listing.
// Keep them here (not as bare literals) so any future change is a one-liner.
const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

func HandleGetPrivate(w http.ResponseWriter, r *http.Request) {
	privateIDStr := r.PathValue("private_id")
	privateID, err := strconv.ParseInt(privateIDStr, 10, 64)
	if err != nil {
		logger.App.Printf("[GET_PRIVATE] error=invalid_private_id raw=%q", privateIDStr)
		utils.JSON(w, http.StatusBadRequest, false, "invalid private_id", nil)
		return
	}

	private, err := models.GetPrivateByID(privateID)
	if err != nil {
		logger.App.Printf("[GET_PRIVATE] error=not_found private_id=%d err=%v", privateID, err)
		utils.JSON(w, http.StatusNotFound, false, "private conversation not found", nil)
		return
	}

	logger.App.Printf("[GET_PRIVATE] success private_id=%d", privateID)
	utils.JSON(w, http.StatusOK, true, "private conversation retrieved successfully", private)
}

func HandleJoinPrivate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxUserID).(int64)
	if !ok {
		logger.App.Printf("[JOIN_PRIVATE] error=missing_user_id_in_context remote=%s", r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized", nil)
		return
	}

	var req struct {
		ReceiverId int64 `json:"receiver_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	defer r.Body.Close()
	if err != nil || req.ReceiverId == 0 {
		logger.App.Printf("[JOIN_PRIVATE] error=invalid_body user_id=%d err=%v", userID, err)
		utils.JSON(w, http.StatusBadRequest, false, "invalid requested data", nil)
		return
	}

	private, err := models.GetPrivateByUsers(userID, req.ReceiverId)
	if err != nil {
		logger.App.Printf("[JOIN_PRIVATE] error=get_private user_id=%d receiver_id=%d err=%v", userID, req.ReceiverId, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to retrieve private conversation", nil)
		return
	}

	if private != nil {
		logger.App.Printf("[JOIN_PRIVATE] existing_private user_id=%d receiver_id=%d private_id=%d", userID, req.ReceiverId, private.ID)
		utils.JSON(w, http.StatusOK, true, "private conversation retrieved successfully", private)
		return
	}

	private, err = models.CreatePrivate(userID, req.ReceiverId)
	if err != nil {
		logger.App.Printf("[JOIN_PRIVATE] error=create_private user_id=%d receiver_id=%d err=%v", userID, req.ReceiverId, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to create private conversation", nil)
		return
	}

	logger.App.Printf("[JOIN_PRIVATE] created_private user_id=%d receiver_id=%d private_id=%d", userID, req.ReceiverId, private.ID)
	utils.JSON(w, http.StatusCreated, true, "private conversation created successfully", private)
}

func HandleGetConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.CtxUserID).(int64)
	if !ok {
		logger.App.Printf("[GET_CONVERSATIONS] error=missing_user_id_in_context remote=%s", r.RemoteAddr)
		utils.JSON(w, http.StatusUnauthorized, false, "Unauthorized", nil)
		return
	}

	privates, err := models.GetPrivatesForUser(userID)
	if err != nil {
		logger.App.Printf("[GET_CONVERSATIONS] error=db user_id=%d err=%v", userID, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to retrieve conversations", nil)
		return
	}

	logger.App.Printf("[GET_CONVERSATIONS] success user_id=%d count=%d", userID, len(privates))
	utils.JSON(w, http.StatusOK, true, "conversations retrieved successfully", privates)
}

func HandleGetPrivateMessages(w http.ResponseWriter, r *http.Request) {
	privateIdStr := r.PathValue("private_id")

	privateID, err := strconv.ParseInt(privateIdStr, 10, 64)
	if err != nil {
		logger.App.Printf("[GET_MESSAGES] error=invalid_private_id raw=%q", privateIdStr)
		utils.JSON(w, http.StatusBadRequest, false, "Invalid private id", nil)
		return
	}

	page := defaultPage
	limit := defaultLimit

	if p := r.URL.Query().Get("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil || page < 1 {
			logger.App.Printf("[GET_MESSAGES] error=invalid_page raw=%q private_id=%d", p, privateID)
			utils.JSON(w, http.StatusBadRequest, false, "invalid page number", nil)
			return
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil || limit < 1 || limit > maxLimit {
			logger.App.Printf("[GET_MESSAGES] error=invalid_limit raw=%q private_id=%d", l, privateID)
			utils.JSON(w, http.StatusBadRequest, false, "invalid limit number (must be between 1 and 100)", nil)
			return
		}
	}

	logger.App.Printf("[GET_MESSAGES] fetching private_id=%d page=%d limit=%d", privateID, page, limit)

	messages, err := models.GetMessagesByPrivateID(privateID, page, limit+1)
	if err != nil {
		logger.App.Printf("[GET_MESSAGES] error=db private_id=%d page=%d limit=%d err=%v", privateID, page, limit, err)
		utils.JSON(w, http.StatusInternalServerError, false, "failed to retrieve messages", nil)
		return
	}

	hasNextPage := false
	if len(messages) > limit {
		hasNextPage = true
		messages = messages[:limit]
	}

	logger.App.Printf("[GET_MESSAGES] success private_id=%d page=%d limit=%d count=%d has_next=%v", privateID, page, limit, len(messages), hasNextPage)
	utils.JSON(w, http.StatusOK, true, "messages retrieved successfully", map[string]any{
		"messages":      messages,
		"page":          page,
		"limit":         limit,
		"has_next_page": hasNextPage,
	})
}
