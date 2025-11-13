// Theme toggle functionality
(function() {
    'use strict';
    
    const THEME_KEY = 'forum-theme';
    const THEME_LIGHT = 'light';
    const THEME_DARK = 'dark';
    
    // Get current theme from localStorage or default to light
    function getCurrentTheme() {
        return localStorage.getItem(THEME_KEY) || THEME_LIGHT;
    }
    
    // Set theme
    function setTheme(theme) {
        if (theme === THEME_DARK) {
            document.documentElement.setAttribute('data-theme', 'dark');
        } else {
            document.documentElement.removeAttribute('data-theme');
        }
        localStorage.setItem(THEME_KEY, theme);
        updateThemeToggleButton(theme);
    }
    
    // Update theme toggle button icon
    function updateThemeToggleButton(theme) {
        const toggleButtons = document.querySelectorAll('.theme-toggle');
        toggleButtons.forEach(button => {
            const icon = button.querySelector('.theme-toggle-icon');
            if (icon) {
                if (theme === THEME_DARK) {
                    icon.textContent = '☀️';
                    button.setAttribute('title', 'Switch to light mode');
                } else {
                    icon.textContent = '🌙';
                    button.setAttribute('title', 'Switch to dark mode');
                }
            }
        });
    }
    
    // Toggle theme
    function toggleTheme() {
        const currentTheme = getCurrentTheme();
        const newTheme = currentTheme === THEME_DARK ? THEME_LIGHT : THEME_DARK;
        setTheme(newTheme);
    }
    
    // Initialize theme on page load
    function initTheme() {
        const savedTheme = getCurrentTheme();
        setTheme(savedTheme);
        
        // Add click handlers to all theme toggle buttons
        document.addEventListener('click', function(e) {
            if (e.target.closest('.theme-toggle')) {
                e.preventDefault();
                toggleTheme();
            }
        });
    }
    
    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTheme);
    } else {
        initTheme();
    }
})();

