// Reaction handler for likes and dislikes
document.addEventListener('DOMContentLoaded', function() {
  document.querySelectorAll('.reaction-btn').forEach(function(btn) {
    btn.addEventListener('click', function(e) {
      e.preventDefault();
      e.stopPropagation();
      const btnElement = this;
      const type = btnElement.getAttribute('data-type');
      const id = btnElement.getAttribute('data-id');
      const like = btnElement.getAttribute('data-like');
      const url = type === 'post' ? '/like_post' : '/like_comment';
      
      console.log('Reaction click:', {type, id, like, url});
      
      fetch(url + '?id=' + id + '&like=' + like, {
        method: 'GET',
        credentials: 'include',
        headers: {
          'Accept': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }
      })
        .then(response => {
          console.log('Response status:', response.status);
          // Проверяем статус 401 (Unauthorized) - перенаправляем на login
          if (response.status === 401) {
            window.location.href = '/login';
            return;
          }
          
          if (!response.ok) {
            // Проверяем Content-Type перед парсингом JSON
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
              return response.json().then(data => {
                // Если ошибка Unauthorized в JSON, тоже перенаправляем
                if (data.error === 'Unauthorized') {
                  window.location.href = '/login';
                  return;
                }
                throw new Error(data.error || 'Request failed');
              });
            } else {
              // Если не JSON, читаем как текст
              return response.text().then(text => {
                throw new Error(text || 'Request failed with status ' + response.status);
              });
            }
          }
          return response.json();
        })
        .then(data => {
          // Проверяем, что data существует (может быть undefined если был редирект)
          if (!data) {
            return;
          }
          
          console.log('Response data:', data);
          if (data.success) {
            // Find the parent container - try different selectors based on page structure
            let actionsContainer = btnElement.closest('.pinterest-card-actions');
            if (!actionsContainer) {
              // Fallback for post.html template
              const container = btnElement.closest('article, .card-body');
              if (container) {
                actionsContainer = container;
              }
            }
            
            if (actionsContainer) {
              // Update likes and dislikes in this container
              const likesSpan = actionsContainer.querySelector('.reaction-btn[data-like="1"] .likes-count');
              const dislikesSpan = actionsContainer.querySelector('.reaction-btn[data-like="0"] .dislikes-count');
              if (likesSpan) likesSpan.textContent = data.likes;
              if (dislikesSpan) dislikesSpan.textContent = data.dislikes;
            }
          } else if (data.error === 'Unauthorized') {
            window.location.href = '/login';
          } else {
            alert('Error: ' + (data.error || 'Unknown error'));
          }
        })
        .catch(error => {
          console.error('Error:', error);
          // Не показываем alert если уже был редирект на login
          if (error.message !== 'Unauthorized' && !error.message.includes('401')) {
            alert('An error occurred: ' + error.message);
          }
        });
      return false;
    });
  });
});

