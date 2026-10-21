/* One request at a time. Only current failed task snapshots are queried. */
(function ($) {
    var indicator = $('#failure-indicator');
    if (!indicator.length) { return; }
    function refresh() {
        if (document.hidden) { window.setTimeout(refresh, 30000); return; }
        $.ajax({
            url: '/failures/summary', dataType: 'json', cache: false, timeout: 10000,
            success: function (data) {
                if (!data || typeof data.count !== 'number' || !$.isArray(data.tasks)) { unavailable(); return; }
                indicator.removeClass('is-unavailable').toggleClass('has-failures', data.count > 0);
                indicator.text(data.count > 0 ? '失败任务 ' + data.count : '暂无失败任务');
                $('#failure-notice').toggleClass('hide', data.count === 0);
                $('#failure-notice-title').text('当前有 ' + data.count + ' 个任务最近执行失败或超时');
                var list = $('#failure-notice-tasks').empty();
                $.each(data.tasks, function (_, task) {
                    var item = $('<li>');
                    $('<a>').attr('href', '/task/logs?id=' + encodeURIComponent(task.id)).text(task.task_name + ' #' + task.id).appendTo(item);
                    $('<span>').text(task.last_status === -2 ? ' · 超时 ' : ' · 失败 ').appendTo(item);
                    $('<span class="failure-summary">').text(task.last_error).attr('title', task.last_error).appendTo(item);
                    list.append(item);
                });
            },
            error: unavailable,
            complete: function () { window.setTimeout(refresh, 30000); }
        });
    }
    function unavailable() {
        indicator.addClass('is-unavailable').text('失败状态暂不可用');
        // Hide stale results rather than report an old count as current.
        $('#failure-notice').addClass('hide');
    }
    $(refresh);
})(jQuery);
