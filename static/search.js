// Simple client-side search
document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    let searchIndex = [];

    // Load index
    // Use absolute path for search.json to work from subdirectories
    const basePath = document.querySelector('h1 a').getAttribute('href'); 
    // Usually "/" or the base URL. If href="/" then "search.json" works if relative to root.
    // Ideally we inject the base URL into a script tag variable, but let's try a robust fetch path.
    
    // Check if we are in a subdirectory
    let fetchPath = 'search.json';
    if (window.location.pathname.includes('/notes/')) {
        // If we are deep in notes, we need to go back to root
        // But cleaner is to use the base tag if it existed, or just absolute path if we know it
        fetchPath = '/search.json';
    }

    fetch(fetchPath)
        .then(response => {
            if (!response.ok) throw new Error("Search index not found");
            return response.json();
        })
        .then(data => {
            searchIndex = data;
        })
        .catch(err => console.log("Search index load error:", err));

    // Handle tag clicks from the document (event delegation)
    // removed old js handler as tags are now real links
    
    searchInput.addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase();
        searchResults.innerHTML = '';
        
        if (query.length < 2) {
            searchResults.style.display = 'none';
            return;
        }

        const results = searchIndex.filter(post => {
            const titleMatch = post.title.toLowerCase().includes(query);
            const contentMatch = post.content.toLowerCase().includes(query);
            
            // Check tags if they exist
            let tagMatch = false;
            if (post.tags) {
                // post.tags is a comma-separated string in our new index
                tagMatch = post.tags.toLowerCase().includes(query);
            }
            
            return titleMatch || contentMatch || tagMatch;
        });

        if (results.length > 0) {
            searchResults.style.display = 'block';
            results.forEach(post => {
                const li = document.createElement('div');
                li.className = 'search-result-item';
                
                let typeLabel = '';
                if (post.type === 'note') {
                    typeLabel = '<span class="tag" style="pointer-events: none;">Note</span>';
                }

                li.innerHTML = `
                    <a href="${post.url}">
                        ${post.title}
                        <div class="meta" style="margin-top:0.2rem; margin-bottom:0;">
                            ${typeLabel} ${post.date ? post.date : ''}
                        </div>
                    </a>
                `;
                searchResults.appendChild(li);
            });
        } else {
            searchResults.style.display = 'none';
        }
    });
    
    // Hide results when clicking outside
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.search-container') && !e.target.classList.contains('tag') && !e.target.classList.contains('topic-tag')) {
            searchResults.style.display = 'none';
        }
    });
});
