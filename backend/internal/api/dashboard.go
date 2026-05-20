package api

import (
	"net/http"

	"movie-tool/backend/internal/catalog"
	"movie-tool/backend/internal/media"
	"movie-tool/backend/internal/scraper"
	"movie-tool/backend/internal/task"
)

type dashboardResponse struct {
	Counts          dashboardCounts                   `json:"counts"`
	Features        []dashboardFeature                `json:"features"`
	RAG             ragConfigResponse                 `json:"rag"`
	RecentTasks     []task.Task                       `json:"recent_tasks"`
	RecentWatchRuns []downloadDirectoryWatchRunDetail `json:"recent_watch_runs"`
}

type dashboardCounts struct {
	Libraries               int `json:"libraries"`
	Media                   int `json:"media"`
	MediaFiles              int `json:"media_files"`
	MissingFiles            int `json:"missing_files"`
	FailedFiles             int `json:"failed_files"`
	DownloadDirectories     int `json:"download_directories"`
	WatchEnabledDirectories int `json:"watch_enabled_directories"`
	Automations             int `json:"automations"`
	EnabledAutomations      int `json:"enabled_automations"`
	OrganizerRules          int `json:"organizer_rules"`
	ScrapeCandidates        int `json:"scrape_candidates"`
	Tasks                   int `json:"tasks"`
	RunningTasks            int `json:"running_tasks"`
	FailedTasks             int `json:"failed_tasks"`
	PendingTasks            int `json:"pending_tasks"`
}

type dashboardFeature struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	libraries, err := s.libraries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items, err := s.catalog.ListItems(r.Context(), catalog.ItemQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	files, err := s.mediaFiles.ListFiles(r.Context(), media.FileQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	directories, err := s.downloads.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	automations, err := s.automations.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rules, err := s.organizer.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	candidates, err := s.scraper.ListCandidates(r.Context(), scraper.CandidateQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	allTasks := s.tasks.List()
	if allTasks == nil {
		allTasks = []task.Task{}
	}
	tasks := newestTasksFirst(append([]task.Task(nil), allTasks...))
	watchRuns := newestTasksFirst(s.tasks.ListByQuery(task.Query{Type: task.TypeDownloadWatch}))
	if len(tasks) > 10 {
		tasks = tasks[:10]
	}
	if tasks == nil {
		tasks = []task.Task{}
	}
	if len(watchRuns) > 5 {
		watchRuns = watchRuns[:5]
	}
	if watchRuns == nil {
		watchRuns = []task.Task{}
	}

	counts := dashboardCounts{
		Libraries:           len(libraries),
		Media:               len(items),
		MediaFiles:          len(files),
		DownloadDirectories: len(directories),
		Automations:         len(automations),
		OrganizerRules:      len(rules),
		ScrapeCandidates:    len(candidates),
		Tasks:               len(allTasks),
	}
	for _, file := range files {
		switch file.Status {
		case media.FileStatusMissing:
			counts.MissingFiles++
		case media.FileStatusFailed:
			counts.FailedFiles++
		}
	}
	for _, directory := range directories {
		if directory.Enabled && directory.WatchEnabled {
			counts.WatchEnabledDirectories++
		}
	}
	for _, automation := range automations {
		if automation.Enabled {
			counts.EnabledAutomations++
		}
	}
	for _, item := range allTasks {
		switch item.Status {
		case task.StatusRunning:
			counts.RunningTasks++
		case task.StatusFailed:
			counts.FailedTasks++
		case task.StatusPending:
			counts.PendingTasks++
		}
	}

	writeJSON(w, http.StatusOK, dashboardResponse{
		Counts:          counts,
		Features:        dashboardFeatures(),
		RAG:             s.ragConfigResponse(),
		RecentTasks:     tasks,
		RecentWatchRuns: s.downloadWatchRunDetails(watchRuns),
	})
}

func dashboardFeatures() []dashboardFeature {
	return []dashboardFeature{
		{Name: "SQLite 与迁移", Status: "已实现", Description: "服务启动时打开 SQLite 并执行内嵌迁移。"},
		{Name: "媒体库 CRUD", Status: "已实现", Description: "媒体库配置已持久化，扫描入口可直接入库。"},
		{Name: "扫描入库", Status: "已实现", Description: "递归扫描媒体文件，解析标题、年份、季集、番号和版本信息。"},
		{Name: "下载目录监听", Status: "已实现", Description: "可配置下载目录，后台轮询稳定文件并触发入库与整理预览。"},
		{Name: "监听历史与重试", Status: "已实现", Description: "记录目录级摘要，支持单批次和近期批次失败目录重试。"},
		{Name: "任务中心", Status: "已实现", Description: "任务、日志、取消与 retry 接口已持久化。"},
		{Name: "自动化管理", Status: "已实现", Description: "自动化规则、运行历史和到期触发已接入 SQL store。"},
		{Name: "刮削候选评分", Status: "已实现", Description: "候选可基于入库文件解析字段自动评分并刷新匹配状态。"},
		{Name: "AV 刮削验证", Status: "已实现", Description: "控制台可解析番号、搜索 JavDB live 候选并显式保存到候选表。"},
		{Name: "文件整理", Status: "已实现", Description: "支持 dry-run、冲突处理、执行、失败修复和回滚。"},
		{Name: "Web 控制台", Status: "本次新增", Description: "服务根路径直接展示当前能力、数据概况和近期任务。"},
		{Name: "本地 RAG 配置", Status: "进行中", Description: "已暴露 oMLX/Ollama、Qdrant 与 collection 配置及健康检查接口。"},
	}
}
