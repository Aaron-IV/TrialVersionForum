// Handle back to top link
document.addEventListener('DOMContentLoaded', function() {
  const backToTopLink = document.getElementById('back-to-top');
  if (backToTopLink) {
    backToTopLink.addEventListener('click', function(e) {
      e.preventDefault();
      window.scrollTo({top: 0, behavior: 'smooth'});
      return false;
    });
  }
});

